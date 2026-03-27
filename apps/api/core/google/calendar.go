package google_calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	authURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL    = "https://oauth2.googleapis.com/token"
	calendarAPI = "https://www.googleapis.com/calendar/v3"
	scope       = "https://www.googleapis.com/auth/calendar.events"
)

type CalendarService struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

func NewCalendarService() *CalendarService {
	return &CalendarService{
		clientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		clientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		redirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}

func (s *CalendarService) IsConfigured() bool {
	return s.clientID != "" && s.clientSecret != "" && s.redirectURL != ""
}

func (s *CalendarService) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", s.clientID)
	params.Set("redirect_uri", s.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("state", state)
	return authURL + "?" + params.Encode()
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
}

func (s *CalendarService) ExchangeCode(code string) (*TokenResponse, error) {
	return s.requestToken(url.Values{
		"code":          {code},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
		"redirect_uri":  {s.redirectURL},
		"grant_type":    {"authorization_code"},
	})
}

func (s *CalendarService) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	return s.requestToken(url.Values{
		"refresh_token": {refreshToken},
		"client_id":     {s.clientID},
		"client_secret": {s.clientSecret},
		"grant_type":    {"refresh_token"},
	})
}

func (s *CalendarService) requestToken(params url.Values) (*TokenResponse, error) {
	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("google token error: %s", token.Error)
	}
	return &token, nil
}

type EventDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type CalendarEvent struct {
	Summary     string        `json:"summary"`
	Location    string        `json:"location,omitempty"`
	Description string        `json:"description,omitempty"`
	Start       EventDateTime `json:"start"`
	End         EventDateTime `json:"end"`
}

type CalendarEventResponse struct {
	ID    string `json:"id"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func BuildEvent(title, location, notes, patientName string, date time.Time, startTime, endTime string) *CalendarEvent {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "UTC"
	}
	dateStr := date.Format("2006-01-02")
	description := ""
	if patientName != "" {
		description = "Paciente: " + patientName
	}
	if notes != "" {
		if description != "" {
			description += "\n"
		}
		description += "Notas: " + notes
	}
	return &CalendarEvent{
		Summary:     title,
		Location:    location,
		Description: description,
		Start:       EventDateTime{DateTime: dateStr + "T" + startTime + ":00", TimeZone: tz},
		End:         EventDateTime{DateTime: dateStr + "T" + endTime + ":00", TimeZone: tz},
	}
}

func (s *CalendarService) CreateEvent(accessToken, calendarID string, event *CalendarEvent) (string, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", calendarAPI+"/calendars/"+url.PathEscape(calendarID)+"/events", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result CalendarEventResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("google calendar API error: %s", result.Error.Message)
	}
	return result.ID, nil
}

func (s *CalendarService) UpdateEvent(accessToken, calendarID, eventID string, event *CalendarEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", calendarAPI+"/calendars/"+url.PathEscape(calendarID)+"/events/"+eventID, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("google calendar update error (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *CalendarService) DeleteEvent(accessToken, calendarID, eventID string) error {
	req, err := http.NewRequest("DELETE", calendarAPI+"/calendars/"+url.PathEscape(calendarID)+"/events/"+eventID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("google calendar delete error (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// TokenExpiry returns the expiry time given ExpiresIn seconds from now.
func TokenExpiry(expiresIn int) time.Time {
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

// IsExpired returns true if the token expires within the next 5 minutes.
func IsExpired(expiry *time.Time) bool {
	if expiry == nil {
		return true
	}
	return expiry.Before(time.Now().Add(5 * time.Minute))
}

// EncodeState encodes a tenant slug into a simple state string.
func EncodeState(tenantSlug string) string {
	return strings.ReplaceAll(tenantSlug, " ", "_")
}

// DecodeState decodes the state back to a tenant slug.
func DecodeState(state string) string {
	return strings.ReplaceAll(state, "_", " ")
}
