package bot

import (
	"encoding/json"

	_ "embed"
)

const (
	UALang = "ua"
)

const (
	userRoleRequestTranslation     = "user_role_request"
	contactPhoneRequestTranslation = "contact_phone_request"

	btnOptionUserRoleSeeker    = "btn_option_user_role_seeker"
	btnOptionUserRoleVolunteer = "btn_option_user_role_volunteer"

	errorChooseOption = "error_choose_option"

	categoryFoodTr       = "category_food"
	categoryMedsTr       = "category_meds"
	categoryClothesTr    = "category_clothes"
	categoryApartmentsTr = "category_apartments"
	categoryThransportTr = "category_transport"
	categoryOtherTr      = "category_other"
)

//go:embed translation.json
var translations []byte // nolint:gochecknoglobals

type Translator interface {
	Translate(key, lang string) string
}

type Tr map[string]map[string]string

func (t Tr) Translate(key, lang string) string {
	return t[key][lang]
}

func NewTranslator() (Tr, error) {
	var trmap = make(map[string]map[string]string)
	return trmap, json.Unmarshal(translations, &trmap)
}
