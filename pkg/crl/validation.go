package crl

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"time"
)

// Validate CRL against the certificate that issued it
func IsCRLValid(crlData *x509.RevocationList, certData *x509.Certificate) (valid bool, nextPublish bool, nextPublishTime time.Time, err error) {
	// Check if the CRL is expired
	isExpired := false
	timeNow := time.Now()
	if crlData.NextUpdate.Before(timeNow) {
		isExpired = true
	}
	// Check if NextPublish is present
	isNextPublishPresent, nextPublishTime, err := isNextPublishPresent(crlData.Extensions)
	if err != nil {
		return false, false, nextPublishTime, fmt.Errorf("ValidateCRL: Error occurred validating NextPublish: %v", err)
	} else if isNextPublishPresent {
		nextPublish = true
	} else {
		nextPublish = false
	}
	// Validate if CRL is signed by the correlating CA certificate
	validSign, err := validateCRLToCertificate(crlData, certData)
	if err != nil {
		return false, false, nextPublishTime, err
	}
	// Final return
	if isExpired {
		return false, nextPublish, nextPublishTime, nil
	} else {
		return validSign, nextPublish, nextPublishTime, nil
	}
}

func validateCRLToCertificate(crlData *x509.RevocationList, certData *x509.Certificate) (bool, error) {
	// Verify the CRL signature using the issuer's public key
	err := crlData.CheckSignatureFrom(certData)
	if err != nil {
		return false, err
	}
	return true, nil
}

func isNextPublishPresent(extensions []pkix.Extension) (bool, time.Time, error) {
	// Extract the Next CRL Publish time from the Microsoft-specific extension
	var nextPublishTime time.Time
	err := findNextPublishExtensionValue(extensions, &nextPublishTime)
	if err != nil {
		return false, nextPublishTime, fmt.Errorf("Error parsing NextCRLPublish extension: %v", err)
	}

	// Prepare the published values for output/usage
	if nextPublishTime.IsZero() {
		nextPublishTime = time.Time{} // Set to zero value if not found
		return false, nextPublishTime, nil
	}
	return true, nextPublishTime, nil
}
