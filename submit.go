package submit

import (
	"log"
	"net/http"
	"os"
)

// Engage func
func Engage(mux http.Handler) error {
	if mux == nil {
		mux = Mux()
	}

	port := os.Getenv("PORT")
	log.Printf("Listening on port %s...\n", port)
	return http.ListenAndServe(":"+port, mux)
}
