package api

import (
	"time"

	"github.com/ImageWare/TLSential/certificate"
	"github.com/ImageWare/TLSential/model"

	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type CertificateHandler interface {
	Get() http.HandlerFunc
	Post() http.HandlerFunc
	Delete() http.HandlerFunc
}

type certHandler struct {
	cs certificate.Service
}

func NewCertificateHandler(cs certificate.Service) CertificateHandler {
	return &certHandler{cs}
}

// CertReq is used for parsing API input
type CertReq struct {
	Domains []string
}

// CertResp is used for exporting User data via API responses
type CertResp struct {
	ID            string
	CommonName    string
	Domains       []string
	CertURL       string
	CertStableURL string
	Expiry        time.Time
	Issued        bool
}

// TODO: Add validation function to make sure domains are actual domains.

// TODO: Refactor sys logging to be more consistent and easier.

// Delete handles all delete calls to api/certificate
func (h *certHandler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// DELETE /api/certificate/
		// Delete all certs
		if id == "" {
			err := h.cs.DeleteAllCerts()
			if err != nil {
				log.Printf("apiCertHandler DELETE, DeleteAllCerts(), %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Delete cert
		u, err := h.cs.Cert(id)
		if err != nil {
			log.Printf("apiCertHandler DELETE, GetCert(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// If it doesn't already exist, return 404.
		if u == nil {
			w.WriteHeader(http.StatusNotFound)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		err = h.cs.DeleteCert(id)
		if err != nil {
			log.Printf("apiCertHandler DELETE, DeleteCert(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

}

func (h *certHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// TODO: Factor out this section into new handler with separate
		// permissions

		// "/api/certificate/"
		if id == "" {
			certs, err := h.cs.AllCerts()
			if err != nil {
				log.Printf("api CertHandler Get(), GetAllCerts(), %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var crs []*CertResp
			for _, c := range certs {
				cr := &CertResp{
					ID:            c.ID,
					CommonName:    c.CommonName,
					Domains:       c.Domains,
					CertURL:       c.CertURL,
					CertStableURL: c.CertStableURL,
					Expiry:        c.Expiry,
					Issued:        c.Issued,
				}
				crs = append(crs, cr)
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")

			err = json.NewEncoder(w).Encode(crs)
			if err != nil {
				log.Printf("apiCertHandler GET ALL, json.Encode(), %s", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		// Return cert if found
		c, err := h.cs.Cert(id)
		if err != nil {
			log.Printf("apiCertHandler GET, GetCert(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if c == nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		// Make an appropriate response object (ie. pkey returned)
		cr := &CertResp{
			ID:            c.ID,
			CommonName:    c.CommonName,
			Domains:       c.Domains,
			CertURL:       c.CertURL,
			CertStableURL: c.CertStableURL,
			Expiry:        c.Expiry,
			Issued:        c.Issued,
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(cr)
		if err != nil {
			log.Printf("apiCertHandler GET, json.Encode(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (h *certHandler) Post() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Body == nil {
			http.Error(w, ErrBodyRequired.Error(), http.StatusBadRequest)
			return
		}

		// Decode JSON payload
		creq := &CertReq{}
		err := json.NewDecoder(r.Body).Decode(creq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Create new Certificate obj.
		c := model.NewCertificate(creq.Domains)
		if err != nil {
			log.Printf("api CertHandler POST, NewCertificate(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Save to database
		err = h.cs.SaveCert(c)
		if err != nil {
			log.Printf("api CertHandler POST, SaveCert(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Build a response obj to return, specifically leaving out
		// Keys and Certs
		cresp := &CertResp{
			ID:            c.ID,
			CommonName:    c.CommonName,
			Domains:       c.Domains,
			CertURL:       c.CertURL,
			CertStableURL: c.CertStableURL,
			Expiry:        c.Expiry,
			Issued:        c.Issued,
		}
		out, err := json.Marshal(cresp)
		if err != nil {
			log.Printf("apiCertHandler POST, json.Marshal(), %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", out)
	}
}