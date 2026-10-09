package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
)

const maxSiteProfileBytes = 64 << 10

var manualSiteFollowUp = []string{
	"Render and review manifests and Helm overlays for the selected profile.",
	"Apply DNS, certificates, ingress and application changes through their existing workflows.",
	"Verify the new domain and TLS before retiring old records or certificates.",
}

type siteSnapshot struct {
	Revision string         `json:"revision"`
	Profile  siteProfile    `json:"profile"`
	Derived  map[string]any `json:"derived"`
}

type sitePlan struct {
	BaseRevision      string       `json:"baseRevision"`
	CandidateRevision string       `json:"candidateRevision"`
	PlanID            string       `json:"planId"`
	ChangedFields     []string     `json:"changedFields"`
	Changes           []siteChange `json:"changes"`
	CurrentDomain     string       `json:"currentDomain"`
	CandidateDomain   string       `json:"candidateDomain"`
	ManualFollowUp    []string     `json:"manualFollowUp"`
}

type siteChange struct {
	Path   string `json:"path"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

func runConfig(args []string, output io.Writer) error {
	if len(args) == 0 {
		return errors.New("config requires snapshot, validate, plan, dry-run, apply, or rollback")
	}
	action := args[0]
	flags := flag.NewFlagSet("render-site config "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	profilePath := flags.String("profile", "", "absolute site profile path")
	candidatePath := flags.String("candidate", "", "absolute candidate profile path")
	confirm := flags.String("confirm", "", "exact plan ID or current revision")
	revision := flags.String("revision", "", "saved revision to restore")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || !filepath.IsAbs(*profilePath) {
		return errors.New("config requires an absolute -profile and valid flags")
	}
	write := func(value any) error { return json.NewEncoder(output).Encode(value) }
	switch action {
	case "snapshot":
		if *candidatePath != "" || *confirm != "" || *revision != "" {
			return errors.New("snapshot accepts only -profile")
		}
		current, raw, derived, err := readSiteProfile(*profilePath)
		if err != nil {
			return err
		}
		return write(siteSnapshot{Revision: siteRevision(raw), Profile: current, Derived: derived})
	case "validate":
		if !filepath.IsAbs(*candidatePath) || *confirm != "" || *revision != "" {
			return errors.New("validate requires an absolute -candidate")
		}
		if _, _, _, err := readSiteProfile(*profilePath); err != nil {
			return err
		}
		_, raw, derived, err := readSiteProfile(*candidatePath)
		if err != nil {
			return err
		}
		return write(map[string]any{"valid": true, "revision": siteRevision(raw), "derived": derived})
	case "plan", "dry-run", "apply":
		if !filepath.IsAbs(*candidatePath) || *revision != "" || (action != "apply" && *confirm != "") || (action == "apply" && *confirm == "") {
			return errors.New("plan/dry-run require -candidate; apply also requires -confirm planId")
		}
		if action == "apply" {
			return withSiteLock(*profilePath, func() error {
				plan, currentRaw, candidateRaw, err := prepareSitePlan(*profilePath, *candidatePath)
				if err != nil {
					return err
				}
				if plan.PlanID != *confirm || len(plan.ChangedFields) == 0 {
					return errors.New("site plan changed or has no changes")
				}
				if err := saveSiteBackup(*profilePath, plan.BaseRevision, currentRaw); err != nil {
					return err
				}
				if err := writeSiteAtomic(*profilePath, candidateRaw); err != nil {
					return err
				}
				return write(map[string]any{"applied": true, "revision": plan.CandidateRevision, "rollbackRevision": plan.BaseRevision, "manualFollowUp": manualSiteFollowUp})
			})
		}
		plan, _, _, err := prepareSitePlan(*profilePath, *candidatePath)
		if err != nil {
			return err
		}
		return write(plan)
	case "rollback":
		if *candidatePath != "" || !validSiteRevision(*revision) || !validSiteRevision(*confirm) {
			return errors.New("rollback requires -revision savedRevision and -confirm currentRevision")
		}
		return withSiteLock(*profilePath, func() error {
			_, currentRaw, _, err := readSiteProfile(*profilePath)
			if err != nil {
				return err
			}
			currentRevision := siteRevision(currentRaw)
			if currentRevision != *confirm {
				return errors.New("site profile changed since rollback was confirmed")
			}
			if currentRevision == *revision {
				return errors.New("site profile already has requested revision")
			}
			_, savedRaw, _, err := readSiteProfile(siteBackupPath(*profilePath, *revision))
			if err != nil || siteRevision(savedRaw) != *revision {
				return errors.New("saved site revision unavailable or invalid")
			}
			if err := saveSiteBackup(*profilePath, currentRevision, currentRaw); err != nil {
				return err
			}
			if err := writeSiteAtomic(*profilePath, savedRaw); err != nil {
				return err
			}
			return write(map[string]any{"rolledBack": true, "revision": *revision, "previousRevision": currentRevision, "manualFollowUp": manualSiteFollowUp})
		})
	default:
		return errors.New("unsupported site config action")
	}
}

func readSiteProfile(path string) (siteProfile, []byte, map[string]any, error) {
	if !filepath.IsAbs(path) {
		return siteProfile{}, nil, nil, errors.New("site profile path must be absolute")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return siteProfile{}, nil, nil, errors.New("site profile unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSiteProfileBytes || info.Mode().Perm()&0o022 != 0 || !siteOwnedByUser(info) {
		return siteProfile{}, nil, nil, errors.New("site profile must be a user-owned regular file and not group/world writable")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxSiteProfileBytes+1))
	if err != nil || len(raw) > maxSiteProfileBytes {
		return siteProfile{}, nil, nil, errors.New("site profile unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var profile siteProfile
	if err := decoder.Decode(&profile); err != nil {
		return siteProfile{}, nil, nil, errors.New("site profile JSON invalid")
	}
	if !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
		return siteProfile{}, nil, nil, errors.New("site profile has trailing data")
	}
	result, err := render(profile, "")
	if err != nil {
		return siteProfile{}, nil, nil, err
	}
	return profile, raw, result.summary, nil
}

func siteOwnedByUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}

func siteRevision(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validSiteRevision(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return false
		}
	}
	return true
}

func prepareSitePlan(currentPath, candidatePath string) (sitePlan, []byte, []byte, error) {
	current, currentRaw, currentDerived, err := readSiteProfile(currentPath)
	if err != nil {
		return sitePlan{}, nil, nil, err
	}
	candidate, candidateRaw, candidateDerived, err := readSiteProfile(candidatePath)
	if err != nil {
		return sitePlan{}, nil, nil, err
	}
	base, next := siteRevision(currentRaw), siteRevision(candidateRaw)
	changes := diffSiteChanges(current, candidate)
	fields := make([]string, 0, len(changes))
	for _, change := range changes {
		fields = append(fields, change.Path)
	}
	plan := sitePlan{BaseRevision: base, CandidateRevision: next, PlanID: siteRevision([]byte(base + ":" + next)), ChangedFields: fields, Changes: changes, CurrentDomain: currentDerived["domain"].(string), CandidateDomain: candidateDerived["domain"].(string), ManualFollowUp: manualSiteFollowUp}
	return plan, currentRaw, candidateRaw, nil
}

func diffSiteChanges(current, candidate siteProfile) []siteChange {
	var left, right map[string]any
	currentJSON, _ := json.Marshal(current)
	candidateJSON, _ := json.Marshal(candidate)
	_ = json.Unmarshal(currentJSON, &left)
	_ = json.Unmarshal(candidateJSON, &right)
	var changed []siteChange
	var walk func(string, map[string]any, map[string]any)
	walk = func(prefix string, a, b map[string]any) {
		keys := make(map[string]bool)
		for key := range a {
			keys[key] = true
		}
		for key := range b {
			keys[key] = true
		}
		for key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			leftMap, leftOK := a[key].(map[string]any)
			rightMap, rightOK := b[key].(map[string]any)
			if leftOK && rightOK {
				walk(path, leftMap, rightMap)
			} else if !reflect.DeepEqual(a[key], b[key]) {
				changed = append(changed, siteChange{Path: path, Before: a[key], After: b[key]})
			}
		}
	}
	walk("", left, right)
	sort.Slice(changed, func(i, j int) bool { return changed[i].Path < changed[j].Path })
	return changed
}

func siteBackupPath(profilePath, revision string) string {
	return profilePath + ".rollback-" + revision
}

func withSiteLock(profilePath string, action func() error) error {
	lock, err := os.OpenFile(profilePath+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return errors.New("site profile lock unavailable")
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.New("site profile is being updated")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return action()
}

func saveSiteBackup(profilePath, revision string, raw []byte) error {
	path := siteBackupPath(profilePath, revision)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o600)
	if errors.Is(err, os.ErrExist) {
		_, existing, _, readErr := readSiteProfile(path)
		if readErr == nil && siteRevision(existing) == revision {
			return nil
		}
		return errors.New("existing site backup does not match revision")
	}
	if err != nil {
		return errors.New("cannot create site rollback revision")
	}
	complete := false
	defer func() {
		file.Close()
		if !complete {
			os.Remove(path)
		}
	}()
	if _, err := file.Write(raw); err != nil {
		return errors.New("cannot write site rollback revision")
	}
	if err := file.Sync(); err != nil {
		return errors.New("cannot sync site rollback revision")
	}
	if err := file.Close(); err != nil {
		return errors.New("cannot close site rollback revision")
	}
	complete = true
	return nil
}

func writeSiteAtomic(profilePath string, raw []byte) error {
	info, err := os.Stat(profilePath)
	if err != nil {
		return errors.New("site profile unavailable")
	}
	file, err := os.CreateTemp(filepath.Dir(profilePath), ".site-config-*")
	if err != nil {
		return errors.New("cannot stage site profile")
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		file.Close()
		return errors.New("cannot protect staged site profile")
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return errors.New("cannot write staged site profile")
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return errors.New("cannot sync staged site profile")
	}
	if err := file.Close(); err != nil {
		return errors.New("cannot close staged site profile")
	}
	if err := os.Rename(file.Name(), profilePath); err != nil {
		return errors.New("cannot replace site profile")
	}
	return nil
}
