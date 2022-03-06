package bot

import (
	"encoding/json"
	"fmt"
	"time"

	_ "embed"
)

const (
	UALang = "UA"

	userRoleRequestTr     = "user_role_request"
	userLocalityRequestTr = "user_locality_request"
	userLocalityReplyTr   = "user_locality_reply"

	seekerCategoryRequestTr           = "seeker_category_request"
	seekerHelpsEmptyTr                = "seeker_helps_empty"
	seekerSubscriptionProposalTr      = "seeker_subscription_proposal"
	seekerLookingForVolunteersTr      = "seeker_looking_for_volunteers"
	seekerSubscriptionCreateSuccessTr = "seeker_subscription_create_success"
	seekerSubscriptionAlreadyExistsTr = "seeker_subscription_already_exists"
	seekerSubscriptionUpdateHeaderTr  = "seeker_subscription_update_header"

	volunteerChosenCategoriesHeaderTr  = "volunteer_chosen_categories_header"
	volunteerChosenCategoriesFooterTr  = "volunteer_chosen_categories_footer"
	volunteerEnterDescriptionRequestTr = "volunteer_enter_description_request"
	volunteerSummaryHeaderTr           = "volunteer_summary_header"
	volunteerSummaryFooterTr           = "volunteer_summary_footer"
	volunteerSelectCategoriesRequestTr = "volunteer_select_categories_request"

	btnOptionRoleSeekerTr    = "btn_option_role_seeker"
	btnOptionUserVolunteerTr = "btn_option_role_volunteer"
	btnOptionNextTr          = "btn_option_next"
	btnOptionSubscribeTr     = "btn_option_subscribe"
	btnOptionDeleteTr        = "bdn_option_delete"

	deleteHelpSuccessTr         = "delete_help_success"
	deleteSubscriptionSuccessTr = "delete_subscription_success"

	errorChooseOptionTr               = "error_choose_option"
	errorPleaseTryAgainTr             = "error_please_try_again"
	errorNoSubscriptionsTr            = "error_no_subscriptions"
	error500Tr                        = "error_500"
	errorHelpsLimitExceededTr         = "error_helps_limit_exceeded"
	errorSubscriptionsLimitExceededTr = "error_subscriptions_limit_exceeded"

	cmdSupportTr                    = "cmd_support"
	cmdStartActivityHeaderTr        = "cmd_start_activity_header"
	cmdStartActivityHelpsTr         = "cmd_start_activity_helps"
	cmdStartActivitySubscriptionsTr = "cmd_start_activity_subscriptions"

	navigationHintTr = "navigation_hint"
)

const (
	weekDaysKey = "week_days"
	monthKey    = "months"
)

//go:embed translation.json
var translations []byte

//go:embed translation_dt.json
var dtTranslations []byte

type Localizer struct {
	textKeys map[string]map[string]string
	timeKeys map[string]map[string][]string
}

func NewLocalizer() (*Localizer, error) {
	var l = new(Localizer)

	var txtMap = make(map[string]map[string]string)
	err := json.Unmarshal(translations, &txtMap)
	if err != nil {
		return nil, err
	}

	var timeMap = make(map[string]map[string][]string)
	err = json.Unmarshal(dtTranslations, &timeMap)
	if err != nil {
		return nil, err
	}

	l.textKeys = txtMap
	l.timeKeys = timeMap
	return l, nil
}

func (l *Localizer) Translate(key, lang string) string { return l.textKeys[key][lang] }

func (l *Localizer) FormatDateTime(t time.Time, lang string) string {
	return fmt.Sprintf("%s %s", l.FormatDate(t, lang), l.FormatTime(t))
}

func (l *Localizer) FormatTime(t time.Time) string {
	return t.Format("15:04")
}

func (l *Localizer) FormatDate(t time.Time, lang string) string {
	return fmt.Sprintf("%s %d %s", l.WeekDay(t.Weekday(), lang), t.Day(), l.Month(t.Month(), lang))
}

func (l *Localizer) Month(month time.Month, lang string) string {
	return l.timeKeys[monthKey][lang][month-1]
}

func (l *Localizer) WeekDay(weekday time.Weekday, lang string) string {
	return l.timeKeys[weekDaysKey][lang][weekday]
}
