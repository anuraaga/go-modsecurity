// Diato - Reverse Proxying for Hipsters
//
// Copyright 2016-2017 Dolf Schimmel
// Copyright (c) 2015 Trustwave Holdings, Inc. (http://www.trustwave.com/)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modsecurity

/*
#cgo CFLAGS: -g -Wall
#cgo LDFLAGS: -lmodsecurity

#include <stdint.h>
#include "modsecurity/modsecurity.h"
#include "modsecurity/transaction.h"

Transaction *msc_new_transaction_cgo(ModSecurity *ms, RulesSet *rules, long logCbData) {
    return msc_new_transaction(ms, rules, (void*)(intptr_t)logCbData);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unsafe"
)

// Represents the inspection on an entire request.
//
// An instance of the transaction struct represents
// an entire request, on its different phases.
type transaction struct {
	// IgnoreRules contains a space-separated list of RulesIDs to be ignored
	IgnoreRules string

	// TransactionBypassed defines if the current transaction has been bypassed.
	// If so, no Logging should be processed, unless ForceLog is true
	TransactionBypassed bool

	// Defines if the Log should be processed even if the Transaction has been
	// bypassed by ignored rules
	ForceLog bool

	// BlockedBy is an array of Rules that caused the transaction to be blocked
	BlockedBy []string

	ruleset     *RuleSet
	msc_txn     *C.struct_Transaction_t
	itemsToFree []unsafe.Pointer
}

var (
	regexIgnoreRule = regexp.MustCompile(`\[id \"(?P<rule>\d*)\"\]`)
)

// NewTransaction Create a new transaction for a given configuration and ModSecurity core.
//
// The transaction is the unit that will be used the inspect every request. It holds
// all the information for a given request.
//
// Remember to cleanup the transaction when the transaction is complete using Cleanup()
func (r *RuleSet) NewTransaction(remoteHost string, remotePort int, localHost string, localPort int) (*transaction, error) {
	msc_txn := C.msc_new_transaction_cgo(r.modsec.modsec, r.msc_rules, C.long(r.modsec.logCallbackId))
	if msc_txn == nil {
		return nil, fmt.Errorf("Could not initialize transaction")
	}

	cRemoteIp := C.CString(remoteHost)
	cLocalIp := C.CString(localHost)

	if C.msc_process_connection(msc_txn, cRemoteIp, C.int(remotePort), cLocalIp, C.int(localPort)) != 1 {
		C.free(unsafe.Pointer(cRemoteIp))
		C.free(unsafe.Pointer(cLocalIp))
		return nil, errors.New("could not process connection")
	}

	txn := &transaction{
		ruleset: r,
		msc_txn: msc_txn,
	}
	txn.deferFree(unsafe.Pointer(cRemoteIp), unsafe.Pointer(cLocalIp))
	return txn, nil
}

// Perform the analysis on the URI and all the query string variables.
//
// There is no direct connection between this function and any phase of
// the SecLanguage's phases. It is something that may occur between the
// SecLanguage phase 1 and 2.
func (txn *transaction) ProcessUri(uri, method, httpVersion string) error {
	cUri := C.CString(uri)
	cMethod := C.CString(method)
	cHttpVersion := C.CString(httpVersion)
	txn.deferFree(
		unsafe.Pointer(cUri),
		unsafe.Pointer(cMethod),
		unsafe.Pointer(cHttpVersion),
	)

	if C.msc_process_uri(txn.msc_txn, cUri, cMethod, cHttpVersion) != 1 {
		return errors.New("Could not process URI")
	}
	return nil
}

// With this function it is possible to feed ModSecurity with a request header.
func (txn *transaction) AddRequestHeader(key, value string) error {
	cKey := C.CString(key)
	cValue := C.CString(value)
	txn.deferFree(unsafe.Pointer(cKey), unsafe.Pointer(cValue))

	if C.msc_add_request_header(txn.msc_txn,
		(*C.uchar)(unsafe.Pointer(cKey)),
		(*C.uchar)(unsafe.Pointer(cValue))) != 1 {
		return errors.New("Could not add request header")
	}
	return nil
}

// This function perform the analysis on the request headers, notice however
// that the headers should be added prior to the execution of this function.
//
// Remember to check for a possible intervention.
func (txn *transaction) ProcessRequestHeaders() error {
	if C.msc_process_request_headers(txn.msc_txn) != 1 {
		return errors.New("Could not process request headers")
	}
	return nil
}

// Adds request body to be inspected.
//
// With this function it is possible to feed ModSecurity with data for
// inspection regarding the request body.
func (txn *transaction) AppendRequestBody(body []byte) error {
	if 1 != C.msc_append_request_body(txn.msc_txn,
		(*C.uchar)(unsafe.Pointer(&body[0])),
		C.size_t(len(body))) {
		return errors.New("Could not append Request Body")
	}

	return nil
}

// Perform the analysis on the request body (if any)
// This function perform the analysis on the request body. It is optional to
// call that function. If this API consumer already know that there isn't a
// body for inspect it is recommended to skip this step.
//
// It is necessary to "append" the request body prior to the execution of this function.
//
// Remember to check for a possible intervention.
func (txn *transaction) ProcessRequestBody() error {
	if C.msc_process_request_body(txn.msc_txn) != 1 {
		return errors.New("Could not process Request Body")
	}

	return nil
}

// With this function it is possible to feed ModSecurity with a response header.
func (txn *transaction) AddResponseHeader(key, value string) error {
	cKey := C.CString(key)
	cValue := C.CString(value)
	txn.deferFree(unsafe.Pointer(cKey), unsafe.Pointer(cValue))

	if C.msc_add_response_header(
		txn.msc_txn,
		(*C.uchar)(unsafe.Pointer(cKey)),
		(*C.uchar)(unsafe.Pointer(cValue))) != 1 {
		return errors.New("Could not add response header")
	}
	return nil
}

// This function perform the analysis on the response headers, notice however
// that the headers should be added prior to the execution of this function.
//
// Remember to check for a possible intervention.
func (txn *transaction) ProcessResponseHeaders(code int, httpVersion string) error {
	cCode := C.int(code)
	cHttpVersion := C.CString(httpVersion)
	if C.msc_process_response_headers(txn.msc_txn,
		cCode,
		(*C.char)(unsafe.Pointer(&cHttpVersion))) != 1 {
		return errors.New("Could not process request headers")
	}
	return nil
}

// Adds response body to be inspected.
//
// With this function it is possible to feed ModSecurity with data for
// inspection regarding the request body.
func (txn *transaction) AppendResponseBody(body []byte) error {
	if 1 != C.msc_append_response_body(txn.msc_txn,
		(*C.uchar)(unsafe.Pointer(&body[0])),
		C.size_t(len(body))) {
		return errors.New("Could not append Response Body")
	}

	return nil
}

// Perform the analysis on the response body (if any)
// This function perform the analysis on the response body. It is optional to
// call that function. If this API consumer already know that there isn't a
// body for inspect it is recommended to skip this step.
//
// It is necessary to "append" the response body prior to the execution of this function.
//
// Remember to check for a possible intervention.
func (txn *transaction) ProcessResponseBody() error {
	if C.msc_process_response_body(txn.msc_txn) != 1 {
		return errors.New("Could not process Response Body")
	}

	return nil
}

// Logging all information relative to this transaction.
//
// At this point there is not need to hold the connection,
// the response can be delivered prior to the execution of
// this method.
func (txn *transaction) ProcessLogging() error {
	if txn.TransactionBypassed {
		return nil
	}
	if C.msc_process_logging(txn.msc_txn) != 1 {
		return errors.New("Could not Process Logging")
	}
	return nil
}

func (txn *transaction) ShouldIntervene() bool {
	intervention := C.struct_ModSecurityIntervention_t{}

	if C.msc_intervention(txn.msc_txn, &intervention) == 0 {
		return false
	}

	// This was the 'better' (but not best) way I've found to ignore rule:
	// Read the log and see if some rule is being ignored.
	// I'm not sorry for this, but I have some shame
	log := C.GoString(intervention.log)
	logRules := regexIgnoreRule.FindStringSubmatch(log)
	if len(logRules) == 2 {
		if txn.IgnoreRules != "" && txn.ShouldIgnore(logRules[1]) {
			txn.TransactionBypassed = true
			return false
		}
		txn.BlockedBy = append(txn.BlockedBy, logRules[1])
	}
	return true
}

// ShouldIgnore returns true if the Intervention should be ignored
// It parses the intervention.log and if the field rule matches with
// something in the IgnoreRules, then return true.
func (txn *transaction) ShouldIgnore(logRule string) bool {
	rules := strings.Split(txn.IgnoreRules, " ")
	for _, rule := range rules {
		if logRule == rule {
			return true
		}
	}
	return false
}

func (txn *transaction) Cleanup() {
	C.msc_transaction_cleanup(txn.msc_txn)
	txn.msc_txn = nil
	for _, freeMe := range txn.itemsToFree {
		C.free(freeMe)
	}
}

func (txn *transaction) deferFree(addToList ...unsafe.Pointer) {
	txn.itemsToFree = append(txn.itemsToFree, addToList...)
}
