package main

import (
	"crypto/x509"
	"encoding/asn1"
	"io"
	"net/http"
	"time"

	pp "github.com/k0kubun/pp"
)

type crlPublishedValues struct {
	ThisUpdate     string `json:"thisUpdate"`
	NextUpdate     string `json:"nextUpdate"`
	NextCRLPublish string `json:"nextPublish"`
}

func main() {

	crlUrl := "http://crl.nhn.no/crl/NHN%20PSKY%20Internal%20CA%20-%20PROD.crl"

	crl, err := retrieveCertificateRevocationList(crlUrl)
	if err != nil {
		pp.Printf("Error retrieving CRL: %v\n", err)
		return
	}

	// crlPublishedValues := crlPublishedValues{
	// 	ThisUpdate:     crl.ThisUpdate.String(),
	// 	NextUpdate:     crl.NextUpdate.String(),
	// 	NextCRLPublish: "", // This is a ADCS (Microsoft) specific field and not part of the standard x509.RevocationList
	// }

	pp.Printf("crl: %+v\n", decodedCrl)
	//pp.Printf("CRL Published Values: %+v\n", crlPublishedValues)

}

func retrieveCertificateRevocationList(url string) (*x509.RevocationList, error) {
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

	// Parse and output the data
	crl, err := x509.ParseRevocationList(data)
	if err != nil {
		return nil, err
	}
	// pp.Printf("Response: %+v\n", crl)

	// crl, error := x509.ParseRevocationList(resp)
	// if error != nil {
	// 	return nil, error
	// }

	return crl, nil
}
