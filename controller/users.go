package controller

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/raminderis/lenslocked/errors"
	"github.com/raminderis/lenslocked/models"
)

type Users struct {
	Templates struct {
		New          Template
		CityTemp     Template
		ShowCityTemp Template
		ContactUs    Template
		ThanksPage   Template
	}
	CityTempS      *models.CityTempS
	EmailService   *models.EmailService
	MessageService *models.MessageService
}

func (u Users) CityTemp(w http.ResponseWriter, r *http.Request) {
	var data struct {
		City string
	}
	data.City = r.FormValue("city")
	u.Templates.CityTemp.Execute(w, r, data)
}

func (u Users) ProcessCityTemp(w http.ResponseWriter, r *http.Request) {
	var data struct {
		City         string
		ApiToken     string
		CityTemp     string
		CityHumidity string
		Time         string
	}
	data.City = r.FormValue("city")
	data.ApiToken = os.Getenv("OPENWEATHER_TOKEN")
	//strings.ToUpper(r.FormValue("city"))
	cityTemp, err := u.CityTempS.Communicate(data.City, data.ApiToken)
	if err != nil {
		fmt.Println(err)
		//http.Error(w, "processing city temp has an error "+fmt.Sprint(err), http.StatusInternalServerError)
		err = errors.Public(err, "CIty Details Unavailable.")
		u.Templates.CityTemp.Execute(w, r, data, err)
		return
	}

	data.City = strings.ToUpper(r.FormValue("city")[:1]) + strings.ToLower(r.FormValue("city")[1:])
	data.CityHumidity = cityTemp.Humidity
	data.CityTemp = cityTemp.Temp
	data.Time = cityTemp.Time
	u.Templates.ShowCityTemp.Execute(w, r, data)
}

func (u Users) ProcessContactUs(w http.ResponseWriter, r *http.Request) {
	//receive message
	var data struct {
		Name    string
		Email   string
		Message string
	}
	data.Name = r.FormValue("name")
	data.Email = r.FormValue("email")
	data.Message = r.FormValue("message")
	message, err := u.MessageService.SaveMessage(data.Name, data.Email, data.Message)
	if err != nil {
		http.Error(w, "we were unable to receive message, try again later.", http.StatusInternalServerError)
	}
	//send a thank you message back
	err = u.EmailService.ThanksMessage(message.Email, message.Message)
	if err != nil {
		http.Error(w, "we were unable to send you a thank you message.", http.StatusInternalServerError)
	}
	u.Templates.ThanksPage.Execute(w, r, data)
}
