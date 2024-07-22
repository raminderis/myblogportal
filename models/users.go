package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ErrEmailTaken = errors.New("models: email address is already in use")
)

type OpenWeatherConfig struct {
	Domain     string
	Path       string
	QueryCity  string
	QueryAppid string
}

type CityTempS struct {
	Temp     string
	Humidity string
	Time     string
}

func DefaultOpenWeatherConfig() OpenWeatherConfig {
	return OpenWeatherConfig{
		Domain: "api.openweathermap.org",
		Path:   "/data/2.5/weather",
	}
}

func OpenWeatherUrlGenerator(city, apiToken string) string {
	urlConfig := DefaultOpenWeatherConfig()
	urlConfig.QueryCity = city
	urlConfig.QueryAppid = apiToken
	return fmt.Sprintf("https://%s%s?q=%s&appid=%s", urlConfig.Domain, urlConfig.Path, urlConfig.QueryCity, urlConfig.QueryAppid)
}

func (us *CityTempS) Communicate(city, apiToken string) (*CityTempS, error) {
	//send query to openweahter
	requestURL := OpenWeatherUrlGenerator(city, apiToken)
	fmt.Println(requestURL)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		fmt.Printf("Communicate: could not create request: %s\n", err)
		return nil, fmt.Errorf("Communicate : %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Communicate: error making http request: %s\n", err)
		return nil, fmt.Errorf("Communicate : %w", err)
	}

	if res.StatusCode != 200 {
		originalErr := errors.New("city details unavailable")
		return nil, fmt.Errorf("Communicate : %v", originalErr)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Communicate: could not read response body: %s\n", err)
		return nil, fmt.Errorf("Communicate : %w", err)
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(resBody, &jsonResponse)
	if err != nil {
		fmt.Printf("Communicate: could not unmarshal response body: %s\n", err)
		return nil, fmt.Errorf("Communicate : %w", err)
	}
	cityTemp := CityTempS{}
	cityTemp.Time = time.Now().Format(time.RFC3339)
	for key, value := range jsonResponse {
		//fmt.Println("Open Weather Response : ", key, value)
		if key == "main" {
			// Type assertion to extract the map[string]interface{} value
			mainData, ok := value.(map[string]interface{})
			if !ok {
				fmt.Println("Error: Unable to assert type for 'main'")
				return nil, fmt.Errorf("Communicate : %w", err)
			}

			// Now you can access specific fields within the 'main' data
			temperature, tempExists := mainData["temp"].(float64)
			if tempExists {
				cityTemp.Temp = fmt.Sprintf("%.2f", temperature-273.15)

			} else {
				fmt.Println("Temperature data not found")
				cityTemp.Temp = "Unknown"
			}

			humidity, humidityExists := mainData["humidity"].(float64)
			if humidityExists {
				cityTemp.Humidity = fmt.Sprintf("%.2f", humidity)

			} else {
				fmt.Println("Humidity data not found")
				cityTemp.Humidity = "Unknown"
			}
			// Handle other fields similarly (e.g., humidity, pressure, etc.)
		}
	}
	// fmt.Println(cityTemp.Temp)
	return &cityTemp, nil
}

type Message struct {
	ID      int
	Name    string
	Email   string
	Message string
}

type MessageService struct {
	DB *sql.DB
}

func (ms *MessageService) SaveMessage(name, email, messagebody string) (*Message, error) {
	message := Message{
		Name:    name,
		Email:   email,
		Message: messagebody,
	}
	row := ms.DB.QueryRow(`
		INSERT INTO messages (sender, email, sender_message)
		VALUES ($1,$2,$3) ON CONFLICT (sender) DO
		UPDATE
		SET email = $2, sender_message = $3 
		RETURNING id`, name, email, messagebody)
	err := row.Scan(&message.ID)
	if err != nil {
		return nil, fmt.Errorf("save message contact service: %w", err)
	}
	return &message, nil
}
