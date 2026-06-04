package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//サービスプリンシパルキーのアクセストークンを返却する
func (c *archiveHandlingServer) issueAccessToken(ctx context.Context) (string, error) {
	assertion, err := c.createSignedJWT()
	if err != nil {
		return "", fmt.Errorf("JWTの生成に失敗しました: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, servicePrincipalTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("トークン発行リクエストの生成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("トークン発行API呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("トークン発行レスポンスの読み取りに失敗しました: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("トークン発行APIがエラーを返却しました: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken    string `json:"access_token"`
		TokenType      string `json:"token_type"`
		TokenExpiredAt string `json:"token_expired_at"`
		ExpiresIn      int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("トークン発行レスポンスのJSON解析に失敗しました: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("トークン発行レスポンスにaccess_tokenがありません")
	}

	return tokenResp.AccessToken, nil
}
//アクセストークン取得のためのJWTを生成する
func (c *archiveHandlingServer) createSignedJWT() (string, error) {
	now := time.Now().UTC().Unix()

	header := map[string]string{
		"alg": "RS256",
		"kid": c.kid,
		"typ": "JWT",
	}
	payload := map[string]any{
		"aud": servicePrincipalTokenEndpoint,
		"exp": now + 300,
		"iat": now,
		"iss": c.servicePrincipalID,
		"sub": c.servicePrincipalID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("JWTヘッダーのJSON化に失敗しました: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("JWTペイロードのJSON化に失敗しました: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	dataToSign := encodedHeader + "." + encodedPayload

	privateKey, err := parseRSAPrivateKey(c.privateKeyPEM)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(dataToSign))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("JWT署名に失敗しました: %w", err)
	}

	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	return dataToSign + "." + encodedSignature, nil
}
//JWT生成のためのRSA秘密鍵をPEM形式から解析する
func parseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(privateKeyPEM), `\\n`, "\n")
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, fmt.Errorf("秘密鍵PEMのデコードに失敗しました")
	}

	if pkcs1Key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return pkcs1Key, nil
	}

	pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("秘密鍵の解析に失敗しました: %w", err)
	}

	rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("RSA秘密鍵ではありません")
	}

	return rsaKey, nil
}
