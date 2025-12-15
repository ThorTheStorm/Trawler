package crl

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"io"
	"net/http"
	"time"
)

// retrieveCertificateRevocationList fetches the CRL from the specified URL
func RetrieveCertificateRevocationList(url string) ([]byte, error) {
	resp, error := http.Get(url)
	if error != nil {
		return nil, error
	}
	// Ensure the response body is closed after reading
	defer resp.Body.Close()

	// Read the raw data to make it usable
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
} // func retrieveCertificateRevocationList

// parseCertificateRevocationList parses the raw CRL data into a structured x509.RevocationList
func ParseCertificateRevocationList(data []byte) (*x509.RevocationList, error) {
	// Parse and output the data
	crl, err := x509.ParseRevocationList(data)
	if err != nil {
		return nil, err
	}
	return crl, nil
} // func parseCertificateRevocationList

func TimeToUpdateCRL(nextUpdate time.Time, nextCRLPublish time.Time, updateThreshold time.Duration) bool {
	if nextCRLPublish.IsZero() {
		// If NextCRLPublish is not available, fall back to NextUpdate
		return time.Until(nextUpdate) <= updateThreshold
	}
	// Use the earlier of NextUpdate and NextCRLPublish
	earliest := nextUpdate
	if nextCRLPublish.Before(nextUpdate) {
		earliest = nextCRLPublish
	}
	return time.Until(earliest) <= updateThreshold
}

// findExtension searches for a specific extension in the list of extensions by its OID
func FindExtension(extensions []pkix.Extension, oid asn1.ObjectIdentifier) *pkix.Extension {
	for _, ext := range extensions {
		if ext.Id.Equal(oid) {
			return &ext
		}
	}
	return nil
} // func findExtension
