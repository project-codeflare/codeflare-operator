package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"

	rayv1api "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	_ "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ServerParameters struct {
	port     int
	certFile string
	keyFile  string
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Config struct {
	Containers []corev1.Container `yaml:"containers"`
	Volumes    []corev1.Volume    `yaml:"volumes"`
}

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	k8sConfig             *rest.Config
	k8sClientSet          *kubernetes.Clientset
	serverParameters      ServerParameters
)

func main() {
	flag.IntVar(&serverParameters.port, "port", 8443, "Webhook server port.")
	flag.StringVar(&serverParameters.certFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&serverParameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()
	
	k8sClientSet = createClientSet()

	// Configure HTTPS server
	http.HandleFunc("/", HandleRoot)
	http.HandleFunc("/mutate", HandleMutate)
	fmt.Print(http.ListenAndServeTLS(":"+strconv.Itoa(serverParameters.port), serverParameters.certFile, serverParameters.keyFile, nil))
}

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("HandleRoot!"))
	if err != nil {
		return
	}
}

func getAdmissionReviewRequest(w http.ResponseWriter, r *http.Request) admissionv1.AdmissionReview {
	requestDump, _ := httputil.DumpRequest(r, true)
	fmt.Printf("Request:\n%s\n", requestDump)

	// Grabbing the http body received on webhook.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Errorf("Error reading webhook request: ", err.Error())
	}

	var admissionReviewReq admissionv1.AdmissionReview

	fmt.Print("deserializing admission review request")
	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = errors.New("malformed admission review: request is nil")
	}

	return admissionReviewReq
}

func HandleMutate(w http.ResponseWriter, r *http.Request) {

	fmt.Print("beginning to handle mutate")
	admissionReviewReq := getAdmissionReviewRequest(w, r)

	var rayCluster rayv1api.RayCluster

	// Deserialize RayCluster from admission review request
	if err := json.Unmarshal(admissionReviewReq.Request.Object.Raw, &rayCluster); err != nil {
		fmt.Errorf("could not unmarshal RayCluster from admission request: %v", err)
	}

	// Extract RayCluster and create patch
	patches, err := createPatch(&rayCluster)
	if err != nil {
		// Handle error
		fmt.Errorf("error creating patch: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Serialize patches into JSON
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		// Handle error
		fmt.Errorf("error marshaling patches: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Log the patches (reflecting the intended mutation)
	fmt.Printf("Generated patches: %s", string(patchBytes))

	// Add patchBytes to the admission response
	admissionReviewResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			UID:     admissionReviewReq.Request.UID,
			Allowed: true,
		},
	}
	admissionReviewResponse.Response.Patch = patchBytes

	// Submit the response
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		fmt.Errorf("marshaling response: %v", err)
	}

	_, err = w.Write(bytes)
	if err != nil {
		return
	}
}
