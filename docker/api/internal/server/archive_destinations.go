package server

import (
	"net/http"
)

type CreateArchiveDestinationRequest struct {
	BucketURI          string `json:"bucket_uri"`
	AwsEndPoint        string `json:"aws_end_point,omitempty"`
	AwsRegion          string `json:"aws_region,omitempty"`
	AwsAccessKeyID     string `json:"aws_access_key_id,omitempty"`
	AwsSecretAccessKey string `json:"aws_secret_access_key,omitempty"`
	GCPCredentialJSON  string `json:"gcp_credential_json,omitempty"`
	AzureAccount       string `json:"azure_account,omitempty"`
	AzureKey           string `json:"azure_key,omitempty"`
}

type CreateArchiveDestinationResponse struct {
	ArchiveDestinationID string `json:"archive_destination_id"`
}

type DeleteArchiveDestinationRequest struct {
	ArchiveDestinationID string `json:"archive_destination_id"`
}

type ArchiveDestination struct {
	ID                  string `json:"id"`
	BucketURI            string `json:"bucket_uri"`
}

type ListArchiveDestinationsResponse struct {
	ArchiveDestinations []ArchiveDestination `json:"archive_destinations"`
}

//アーカイブ保存先作成（必須項目は保存先クラウドに依存するため、バケットURI有無のみ検証）
func (s *Server) createArchiveDestination(w http.ResponseWriter, r *http.Request) {
	var req CreateArchiveDestinationRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}
	if !s.requireNonEmptyString(w, req.BucketURI, "bucket_uri") {
		return
	}

	if err := s.callImageFluxAPI(w, r, imagefluxCreateArchiveDestinationTarget, req); err != nil {
		s.writeImageFluxError(w, err)
	}
}

//アーカイブ保存先削除
func (s *Server) deleteArchiveDestination(w http.ResponseWriter, r *http.Request) {
	var req DeleteArchiveDestinationRequest
	if !s.decodeJSONBody(w, r, &req) {
		return
	}
	if !s.requireNonEmptyString(w, req.ArchiveDestinationID, "archive_destination_id") {
		return
	}

	if err := s.callImageFluxAPI(w, r, imagefluxDeleteArchiveDestinationTarget, req); err != nil {
		s.writeImageFluxError(w, err)
	}
}

//アーカイブ保存先一覧取得
func (s *Server) listArchiveDestinations(w http.ResponseWriter, r *http.Request) {
	if err := s.callImageFluxAPI(w, r, imagefluxListArchiveDestinationsTarget, map[string]any{}); err != nil {
		s.writeImageFluxError(w, err)
	}
}
