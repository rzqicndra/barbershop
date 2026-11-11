package app

import (
	"net/http"
	"time"
)

type Service struct {
	Category    string `json:"category"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	Duration    int    `json:"duration"`
	Active      bool   `json:"active"`
	Version     int    `json:"version"`
}

type Barber struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Specialty  string   `json:"specialty"`
	ServiceIDs []string `json:"serviceIds"`
	Start      int      `json:"start"`
	End        int      `json:"end"`
	BreakStart int      `json:"breakStart"`
	BreakEnd   int      `json:"breakEnd"`
	DaysOff    []int    `json:"daysOff"`
	Active     bool     `json:"active"`
	Version    int      `json:"version"`
}

type TimeOff struct {
	ID       string `json:"id"`
	BarberID string `json:"barberId"`
	Date     string `json:"date"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Reason   string `json:"reason"`
}

type Profile struct {
	MapsURL            string   `json:"mapsUrl"`
	MapsEmbedURL       string   `json:"mapsEmbedUrl"`
	InstagramURL       string   `json:"instagramUrl"`
	WhatsApp           string   `json:"whatsapp"`
	Wordmark           string   `json:"wordmark"`
	LogoImageURL       string   `json:"logoImageUrl"`
	HeroEyebrow        string   `json:"heroEyebrow"`
	HeroTitle          string   `json:"heroTitle"`
	HeroDescription    string   `json:"heroDescription"`
	HeroImageURL       string   `json:"heroImageUrl"`
	FooterTagline      string   `json:"footerTagline"`
	GalleryTitle       string   `json:"galleryTitle"`
	GalleryDescription string   `json:"galleryDescription"`
	Name               string   `json:"name"`
	Location           string   `json:"location"`
	Open               int      `json:"open"`
	Close              int      `json:"close"`
	ClosedDates        []string `json:"closedDates"`
	Version            int      `json:"version"`
}

type Catalog struct {
	Gallery  []GalleryPhoto `json:"gallery"`
	FAQs     []FAQ          `json:"faqs"`
	Reviews  []Review       `json:"reviews"`
	Rating   RatingSummary  `json:"rating"`
	Profile  Profile        `json:"profile"`
	Services []Service      `json:"services"`
	Barbers  []Barber       `json:"barbers"`
	TimeOff  []TimeOff      `json:"timeOff"`
}

type History struct {
	Action string `json:"action"`
	By     string `json:"by"`
	At     int64  `json:"at"`
	Detail string `json:"detail,omitempty"`
}

type Booking struct {
	BarberName  string    `json:"barberName"`
	Review      *Review   `json:"review,omitempty"`
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Date        string    `json:"date"`
	Start       int       `json:"start"`
	ServiceID   string    `json:"serviceId"`
	ServiceName string    `json:"serviceName"`
	Duration    int       `json:"duration"`
	Price       int       `json:"price"`
	BarberID    string    `json:"barberId"`
	Customer    string    `json:"customer"`
	Phone       string    `json:"phone,omitempty"`
	Source      string    `json:"source"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	Token       string    `json:"token,omitempty"`
	History     []History `json:"history,omitempty"`
	startsAt    time.Time
	endsAt      time.Time
	tokenNonce  string
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	BarberID string `json:"barberId,omitempty"`
	Active   bool   `json:"active"`
}

type problem struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (p *problem) Error() string { return p.Message }

func invalid(message string) error {
	return &problem{Status: http.StatusUnprocessableEntity, Code: "INVALID_INPUT", Message: message}
}

func conflict(code, message string) error {
	return &problem{Status: http.StatusConflict, Code: code, Message: message}
}

type GalleryPhoto struct {
	ID               string `json:"id"`
	ImageURL         string `json:"imageUrl"`
	Title            string `json:"title"`
	Caption          string `json:"caption"`
	AltText          string `json:"altText"`
	SortOrder        int    `json:"sortOrder"`
	Active           bool   `json:"active"`
	ConsentConfirmed bool   `json:"consentConfirmed"`
	Version          int    `json:"version"`
}
type FAQ struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	SortOrder int    `json:"sortOrder"`
	Active    bool   `json:"active"`
	Version   int    `json:"version"`
}
type Review struct {
	ID          string    `json:"id"`
	Rating      int       `json:"rating"`
	DisplayName string    `json:"displayName"`
	Comment     string    `json:"comment"`
	ServiceName string    `json:"serviceName"`
	BarberName  string    `json:"barberName"`
	CreatedAt   time.Time `json:"createdAt"`
}
type RatingSummary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}
type MediaAsset struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int    `json:"size"`
}
