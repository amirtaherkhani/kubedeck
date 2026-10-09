package management

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func testManager(t *testing.T) *Manager {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	discovery := kubefake.NewSimpleClientset().Discovery().(*discoveryfake.FakeDiscovery)
	discovery.Resources = []*metav1.APIResourceList{{GroupVersion: "v1", APIResources: []metav1.APIResource{{Name: "configmaps", Kind: "ConfigMap", Namespaced: true, Verbs: metav1.Verbs{"get", "list", "create", "update", "patch", "delete", "watch"}}, {Name: "configmaps/status", Kind: "ConfigMap", Namespaced: true, Verbs: metav1.Verbs{"get"}}, {Name: "secrets", Kind: "Secret", Namespaced: true}, {Name: "secrets/status", Kind: "Secret", Namespaced: true, Verbs: metav1.Verbs{"get"}}}}}
	item := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "sample", "namespace": "apps", "resourceVersion": "4", "uid": "uid-1"}, "data": map[string]any{"a": "b"}}}
	dyn := dynamicfake.NewSimpleDynamicClient(scheme, item)
	return &Manager{Dynamic: dyn, Discovery: discovery, Kube: kubefake.NewSimpleClientset(), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}
func request(m *Manager, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	out := httptest.NewRecorder()
	m.Handler().ServeHTTP(out, req)
	return out
}
func TestResourceReadAndSecretDenial(t *testing.T) {
	m := testManager(t)
	response := request(m, "GET", "/v1/manage/resources/core/v1/configmaps?namespace=apps&name=sample", nil, nil)
	if response.Code != 200 {
		t.Fatalf("get status %d: %s", response.Code, response.Body.String())
	}
	var data map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil || data["kind"] != "ConfigMap" {
		t.Fatalf("body: %s", response.Body.String())
	}
	response = request(m, "GET", "/v1/manage/resources/core/v1/secrets?namespace=apps&name=secret", nil, nil)
	if response.Code != 400 {
		t.Fatalf("secret status %d", response.Code)
	}
}
func TestDeleteRequiresUIDAndExactConfirmation(t *testing.T) {
	m := testManager(t)
	path := "/v1/manage/resources/core/v1/configmaps?namespace=apps&name=sample"
	response := request(m, "DELETE", path, nil, nil)
	if response.Code != 409 {
		t.Fatalf("unguarded delete status %d", response.Code)
	}
	response = request(m, "DELETE", path, nil, map[string]string{"If-Match-UID": "uid-1", "X-KuchDesk-Confirm": "apps/other"})
	if response.Code != 409 {
		t.Fatalf("wrong confirmation status %d", response.Code)
	}
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	if _, err := m.Dynamic.Resource(gvr).Namespace("apps").Get(httptest.NewRequest("GET", "/", nil).Context(), "sample", metav1.GetOptions{}); err != nil {
		t.Fatal("object was deleted")
	}
}
func TestUpdateRequiresResourceVersion(t *testing.T) {
	m := testManager(t)
	path := "/v1/manage/resources/core/v1/configmaps?namespace=apps&name=sample"
	body := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"sample","namespace":"apps"}}`)
	response := request(m, "PUT", path, body, nil)
	if response.Code != 409 {
		t.Fatalf("unguarded update status %d", response.Code)
	}
}

func TestApplyAndPatchRequireVersionAndConfirmation(t *testing.T) {
	m := testManager(t)
	m.Dynamic.(*dynamicfake.FakeDynamicClient).PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patch := action.(k8stesting.PatchAction).GetPatch()
		if !bytes.Contains(patch, []byte(`"resourceVersion":"4"`)) {
			t.Errorf("missing version precondition: %s", patch)
		}
		return true, &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "sample", "namespace": "apps"}}}, nil
	})
	path := "/v1/manage/resources/core/v1/configmaps?namespace=apps&name=sample&type=apply&fieldManager=kuchdesk"
	body := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"sample","namespace":"apps"},"data":{"a":"new"}}`)
	response := request(m, "PATCH", path, body, map[string]string{"If-Match": "4"})
	if response.Code != 409 {
		t.Fatalf("unconfirmed apply status %d", response.Code)
	}
	response = request(m, "PATCH", path, body, map[string]string{"If-Match": "4", "X-KuchDesk-Confirm": "apps/sample"})
	if response.Code != 200 {
		t.Fatalf("apply status %d: %s", response.Code, response.Body.String())
	}
}
func TestCapabilitiesExcludeSecrets(t *testing.T) {
	m := testManager(t)
	response := request(m, "GET", "/v1/manage/capabilities", nil, nil)
	if response.Code != 200 {
		t.Fatalf("capabilities status %d", response.Code)
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"resource":"secrets"`)) {
		t.Fatal("Secret resource advertised")
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"resource":"configmaps"`)) {
		t.Fatalf("ConfigMap missing: %s", response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"resource":"configmaps/status"`)) || bytes.Contains(response.Body.Bytes(), []byte(`"resource":"secrets/status"`)) {
		t.Fatalf("subresource filtering incorrect: %s", response.Body.String())
	}
}

func TestCapabilitiesCheckNamespaceAndSubresourceAuthorization(t *testing.T) {
	m := testManager(t)
	var checked []authv1.ResourceAttributes
	m.Kube.(*kubefake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		attrs := *action.(k8stesting.CreateAction).GetObject().(*authv1.SelfSubjectAccessReview).Spec.ResourceAttributes
		checked = append(checked, attrs)
		return true, &authv1.SelfSubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: attrs.Verb == "get"}}, nil
	})
	response := request(m, "GET", "/v1/manage/capabilities?namespace=apps", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("capabilities status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Resources []struct {
			Resource     string          `json:"resource"`
			AllowedVerbs map[string]bool `json:"allowedVerbs"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, res := range body.Resources {
		if res.Resource == "configmaps/status" {
			found = true
			if !res.AllowedVerbs["get"] {
				t.Fatal("authorized status read omitted")
			}
		}
		if res.Resource == "configmaps" && res.AllowedVerbs["delete"] {
			t.Fatal("denied delete advertised as allowed")
		}
	}
	if !found {
		t.Fatal("status subresource missing")
	}
	var checkedStatus bool
	for _, attrs := range checked {
		if attrs.Namespace != "apps" || attrs.Resource != "configmaps" || attrs.Group != "" || attrs.Version != "v1" {
			t.Fatalf("incorrect review attributes: %+v", attrs)
		}
		if attrs.Subresource == "status" && attrs.Verb == "get" {
			checkedStatus = true
		}
	}
	if !checkedStatus {
		t.Fatal("no status subresource authorization review")
	}
}

func TestCapabilitiesWithoutNamespaceDoNotInferNamespacedAccess(t *testing.T) {
	m := testManager(t)
	m.Kube.(*kubefake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		t.Fatal("unexpected authorization review without namespace")
		return true, nil, nil
	})
	response := request(m, "GET", "/v1/manage/capabilities", nil, nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"allowedVerbs":{}`)) {
		t.Fatalf("capabilities status %d: %s", response.Code, response.Body.String())
	}
	if response := request(m, "GET", "/v1/manage/capabilities?namespace=invalid_name", nil, nil); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid namespace status %d", response.Code)
	}
}

func TestReadWorkloadStatusAndScale(t *testing.T) {
	m := testManager(t)
	m.Kube.(*kubefake.Clientset).PrependReactor("get", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() == "scale" {
			return true, &autoscalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "apps", ResourceVersion: "7"}, Spec: autoscalingv1.ScaleSpec{Replicas: 3}}, nil
		}
		return true, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "apps"}, Status: appsv1.DeploymentStatus{AvailableReplicas: 2}}, nil
	})
	status := request(m, "GET", "/v1/manage/workloads/deployments/apps/sample/status", nil, nil)
	if status.Code != http.StatusOK || !bytes.Contains(status.Body.Bytes(), []byte(`"availableReplicas":2`)) {
		t.Fatalf("status read %d: %s", status.Code, status.Body.String())
	}
	scale := request(m, "GET", "/v1/manage/workloads/deployments/apps/sample/scale", nil, nil)
	if scale.Code != http.StatusOK || !bytes.Contains(scale.Body.Bytes(), []byte(`"replicas":3`)) || !bytes.Contains(scale.Body.Bytes(), []byte(`"resourceVersion":"7"`)) {
		t.Fatalf("scale read %d: %s", scale.Code, scale.Body.String())
	}
	if response := request(m, "GET", "/v1/manage/workloads/daemonsets/apps/sample/scale", nil, nil); response.Code != http.StatusBadRequest {
		t.Fatalf("DaemonSet scale status %d", response.Code)
	}
}

func TestCreateRequiresExactConfirmation(t *testing.T) {
	m := testManager(t)
	path := "/v1/manage/resources/core/v1/configmaps?namespace=apps"
	body := []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"new","namespace":"apps"}}`)
	response := request(m, "POST", path, body, nil)
	if response.Code != 409 {
		t.Fatalf("unconfirmed create status %d", response.Code)
	}
	response = request(m, "POST", path, body, map[string]string{"X-KuchDesk-Confirm": "apps/other"})
	if response.Code != 409 {
		t.Fatalf("wrong create confirmation status %d", response.Code)
	}
}

func TestMergePatchRejectsNullDocument(t *testing.T) {
	m := testManager(t)
	path := "/v1/manage/resources/core/v1/configmaps?namespace=apps&name=sample"
	response := request(m, "PATCH", path, []byte("null"), map[string]string{"If-Match": "4", "X-KuchDesk-Confirm": "apps/sample"})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("null patch status %d: %s", response.Code, response.Body.String())
	}
}

func TestScaleRequiresExplicitReplicas(t *testing.T) {
	m := testManager(t)
	response := request(m, "POST", "/v1/manage/workloads/deployments/apps/sample/scale", []byte(`{"resourceVersion":"4"}`), map[string]string{"X-KuchDesk-Confirm": "deployments/apps/sample"})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing replicas status %d: %s", response.Code, response.Body.String())
	}
}
