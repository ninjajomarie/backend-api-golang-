package external

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/lib/pq/hstore"
	"github.com/shopspring/decimal"
)

type WaitlistEmailVars struct {
	WaitlistCode string
}

type EmailType string

const (
	EmailType_Waitlist       EmailType = "WAITLIST"
	EmailType_ForgotPassword EmailType = "FORGOT_PASSWORD"
)

func (w EmailType) String() string {
	switch w {
	case EmailType_Waitlist:
		return "WAITLIST"
	case EmailType_ForgotPassword:
		return "FORGOT_PASSWORD"
	}
	return ""
}

func (e *EmailType) Scan(src interface{}) error {
	var srcStr string
	switch src.(type) {
	case string:
		srcStr = src.(string)
	case []byte:
		srcStr = string(src.([]byte))
	default:
		return fmt.Errorf("Scan: no enum str for %v", src)
	}

	switch srcStr {

	case "WAITLIST":
		*e = EmailType_Waitlist

	case "FORGOT_PASSWORD":
		*e = EmailType_ForgotPassword

	default:
		return fmt.Errorf("Scan: no enum str for %v", src)
	}
	return nil
}

func (e EmailType) Value() (driver.Value, error) {
	switch e {

	case EmailType_Waitlist:
		return driver.Value("WAITLIST"), nil

	case EmailType_ForgotPassword:
		return driver.Value("FORGOT_PASSWORD"), nil

	default:
		return driver.Value(""), fmt.Errorf("Value: no val for %v", e)
	}
}

type EmailStatus string

const (
	EmailStatus_Pending    EmailStatus = "PENDING"
	EmailStatus_Processing EmailStatus = "PROCESSING"
	EmailStatus_Sent       EmailStatus = "SENT"
)

func (w EmailStatus) String() string {
	switch w {
	case EmailStatus_Pending:
		return "PENDING"
	case EmailStatus_Processing:
		return "PROCESSING"
	case EmailStatus_Sent:
		return "SENT"
	}
	return ""
}

func (e *EmailStatus) Scan(src interface{}) error {
	var srcStr string
	switch src.(type) {
	case string:
		srcStr = src.(string)
	case []byte:
		srcStr = string(src.([]byte))
	default:
		return fmt.Errorf("Scan: no enum str for %v", src)
	}

	switch srcStr {

	case "PENDING":
		*e = EmailStatus_Pending

	case "PROCESSING":
		*e = EmailStatus_Processing

	case "SENT":
		*e = EmailStatus_Sent

	default:
		return fmt.Errorf("Scan: no enum str for %v", src)
	}
	return nil
}

func (e EmailStatus) Value() (driver.Value, error) {
	switch e {

	case EmailStatus_Pending:
		return driver.Value("PENDING"), nil

	case EmailStatus_Processing:
		return driver.Value("PROCESSING"), nil

	case EmailStatus_Sent:
		return driver.Value("SENT"), nil

	default:
		return driver.Value(""), fmt.Errorf("Value: no val for %v", e)
	}
}

type HStoreMap map[string]string

func (h *HStoreMap) Scan(src interface{}) error {
	var out hstore.Hstore
	if err := out.Scan(src); err != nil {
		return err
	}

	*h = make(map[string]string)
	for key, val := range out.Map {
		if val.Valid {
			(*h)[key] = val.String
		}
	}
	return nil
}

func (h HStoreMap) Value() (driver.Value, error) {
	var hs hstore.Hstore
	hs.Map = make(map[string]sql.NullString)
	for k, v := range h {
		hs.Map[k] = sql.NullString{
			Valid:  true,
			String: v,
		}
	}
	return hs.Value()
}

type SocialNetwork string

const (
	SocialNetwork_Unknown   SocialNetwork = "UNKNOWN"
	SocialNetwork_Facebook  SocialNetwork = "FACEBOOK"
	SocialNetwork_Instagram SocialNetwork = "INSTAGRAM"
	SocialNetwork_Twitter   SocialNetwork = "TWITTER"
	SocialNetwork_Twitch    SocialNetwork = "TWITCH"
)

func (w SocialNetwork) String() string {
	switch w {
	case SocialNetwork_Facebook:
		return "FACEBOOK"
	case SocialNetwork_Instagram:
		return "INSTAGRAM"
	case SocialNetwork_Twitter:
		return "TWITTER"
	case SocialNetwork_Twitch:
		return "TWITCH"
	}
	return "UNKNOWN"
}

func (e *SocialNetwork) Scan(src interface{}) error {
	var srcStr string
	switch src.(type) {
	case string:
		srcStr = src.(string)
	case []byte:
		srcStr = string(src.([]byte))
	default:
		return fmt.Errorf("Scan: no enum str for %v", src)
	}

	switch srcStr {

	case "FACEBOOK":
		*e = SocialNetwork_Facebook

	case "INSTAGRAM":
		*e = SocialNetwork_Instagram

	case "TWITTER":
		*e = SocialNetwork_Twitter

	case "TWITCH":
		*e = SocialNetwork_Twitch

	case "UNKNOWN":
		*e = SocialNetwork_Unknown

	default:
		return fmt.Errorf("Scan: no enum str for %v", src)
	}
	return nil
}

func (e SocialNetwork) Value() (driver.Value, error) {
	switch e {

	case SocialNetwork_Facebook:
		return driver.Value("FACEBOOK"), nil

	case SocialNetwork_Instagram:
		return driver.Value("INSTAGRAM"), nil

	case SocialNetwork_Twitter:
		return driver.Value("TWITTER"), nil

	default:
		return driver.Value(""), fmt.Errorf("Value: no val for %v", e)
	}
}

type Email struct {
	ID           int           `json:"id,omitempty"`
	UserID       sql.NullInt64 `json:"user_id,omitempty"`
	EmailAddress string        `json:"email_address,omitempty"`
	TemplateName string        `json:"template_name,omitempty"`
	TemplateVars HStoreMap     `json:"template_vars,omitempty"`
	Type         EmailType     `json:"type,omitempty"`
	Status       EmailStatus   `json:"status,omitempty"`
	CreatedAt    pq.NullTime   `json:"created_at,omitempty"`
	UpdatedAt    pq.NullTime   `json:"updated_at,omitempty"`
	SentAt       pq.NullTime   `json:"sent_at,omitempty"`
}

type VideoDetails struct {
	ID        *NullInt64   `json:"id,omitempty"`
	FileID    *NullInt64   `json:"file_id,omitempty"`
	Duration  *NullDecimal `json:"duration,omitempty"`
	UpdatedAt *NullTime    `json:"updated_at,omitempty"`
	CreatedAt *NullTime    `json:"created_at,omitempty"`
}

type File struct {
	ID           *NullInt64    `json:"id,omitempty"`
	UserID       *NullInt64    `json:"user_id,omitempty"`
	Description  *NullString   `json:"description,omitempty"`
	Extension    *NullString   `json:"extension,omitempty"`
	Name         *NullString   `json:"name,omitempty"`
	Size         *NullInt64    `json:"size,omitempty"`
	Type         *NullString   `json:"type,omitempty"`
	IsActive     *NullBool     `json:"is_active,omitempty"`
	UpdatedAt    *NullTime     `json:"updated_at,omitempty"`
	CreatedAt    *NullTime     `json:"created_at,omitempty"`
	VideoDetails *VideoDetails `json:"video_details,omitempty"`
}

type ModuleBanner struct {
	ID        int       `json:"id,omitempty"`
	ModuleID  int       `json:"module_id,omitempty"`
	FileID    int       `json:"file_id,omitempty"`
	File      *File     `json:"file,omitempty"`
	CreatedAt *NullTime `json:"created_at,omitempty"`
	UpdatedAt *NullTime `json:"updated_at,omitempty"`
}

type ModuleFile struct {
	ID        int       `json:"id,omitempty"`
	ModuleID  int       `json:"module_id,omitempty"`
	Ranking   int       `json:"ranking,omitempty"`
	FileID    string    `json:"file_id,omitempty"`
	File      *File     `json:"file,omitempty"`
	CreatedAt *NullTime `json:"created_at,omitempty"`
	UpdatedAt *NullTime `json:"updated_at,omitempty"`
}

type ModuleCategory struct {
	ID          int       `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   *NullTime `json:"created_at,omitempty"`
	UpdatedAt   *NullTime `json:"updated_at,omitempty"`
}

type ModuleSupportingMaterial struct {
	ID          int       `json:"id,omitempty"`
	ModuleID    int       `json:"module_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Url         string    `json:"url,omitempty"`
	IsActive    bool      `json:"is_active,omitempty"`
	CreatedAt   *NullTime `json:"created_at,omitempty"`
	UpdatedAt   *NullTime `json:"updated_at,omitempty"`
}

type ModuleLearningOutcome struct {
	ID          int       `json:"id,omitempty"`
	ModuleID    int       `json:"module_id,omitempty"`
	Description string    `json:"description,omitempty"`
	Ranking     int       `json:"ranking,omitempty"`
	IsActive    bool      `json:"is_active,omitempty"`
	CreatedAt   *NullTime `json:"created_at,omitempty"`
	UpdatedAt   *NullTime `json:"updated_at,omitempty"`
}

type QuestionOption struct {
	ID             int       `json:"id,omitempty"`
	QuizQuestionID int       `json:"quiz_question_id,omitempty"`
	Name           string    `json:"name,omitempty"`
	Description    string    `json:"description,omitempty"`
	Ranking        int       `json:"ranking,omitempty"`
	CreatedAt      *NullTime `json:"created_at,omitempty"`
	UpdatedAt      *NullTime `json:"updated_at,omitempty"`
}

type Answer struct {
	QuestionID    int `json:"question_id,omitempty"`
	AnswerRanking int `json:"answer_ranking,omitempty"`
}

type Question struct {
	ID                  int               `json:"id,omitempty"`
	QuizID              int               `json:"quiz_id,omitempty"`
	Name                string            `json:"name,omitempty"`
	Description         string            `json:"description,omitempty"`
	Ranking             int               `json:"ranking,omitempty"`
	AnswerOptionRanking int               `json:"answer_option_ranking,omitempty"`
	CreatedAt           *NullTime         `json:"created_at,omitempty"`
	UpdatedAt           *NullTime         `json:"updated_at,omitempty"`
	Options             []*QuestionOption `json:"options,omitempty"`
}

type Quiz struct {
	ID           int             `json:"id,omitempty"`
	ModuleID     int             `json:"module_id,omitempty"`
	Name         string          `json:"name,omitempty"`
	PassingGrade decimal.Decimal `json:"passing_grade,omitempty"`
	Description  string          `json:"description,omitempty"`
	IsActive     bool            `json:"is_active,omitempty"`
	CreatedAt    *NullTime       `json:"created_at,omitempty"`
	UpdatedAt    *NullTime       `json:"updated_at,omitempty"`
	Questions    []*Question     `json:"questions,omitempty"`
}

type Module struct {
	ID                 int                         `json:"id,omitempty"`
	UserID             int                         `json:"user_id,omitempty"`
	Name               string                      `json:"name,omitempty"`
	CategoryID         int                         `json:"category_id,omitempty"`
	Description        string                      `json:"description,omitempty"`
	Latitude           *NullDecimal                `json:"latitude,omitempty"`
	Longitude          *NullDecimal                `json:"longitude,omitempty"`
	Hashtags           string                      `json:"hashtags,omitempty"`
	Ranking            int                         `json:"ranking,omitempty"`
	Free               bool                        `json:"free,omitempty"`
	IsActive           bool                        `json:"is_active,omitempty"`
	CreatedAt          *NullTime                   `json:"created_at,omitempty"`
	UpdatedAt          *NullTime                   `json:"updated_at,omitempty"`
	Banner             *ModuleBanner               `json:"module_banner,omitempty"`
	Quizzes            []*Quiz                     `json:"quizzes,omitempty"`
	Category           *ModuleCategory             `json:"module_category,omitempty"`
	Files              []*ModuleFile               `json:"module_files,omitempty"`
	LearningOutcomes   []*ModuleLearningOutcome    `json:"moduel_learning_outcomes,omitempty"`
	SupportingMaterial []*ModuleSupportingMaterial `json:"module_supporting_material,omitempty"`
}

type WaitlistItem struct {
	ID                     int        `json:"id,omitempty"`
	EmailAddress           string     `json:"email_address,omitempty"`
	OwnerWaitlistCode      string     `json:"owner_waitlist_code,omitempty"`
	OriginalReferralCodeID *int       `json:"original_referral_code_id,omitempty"`
	OriginalWaitlistCodeID *int       `json:"original_waitlist_code_id,omitempty"`
	CreatedAt              *time.Time `json:"created_at,omitempty"`
	UpdatedAt              *time.Time `json:"updated_at,omitempty"`
}

type WaitlistRequest struct {
	EmailAddress         string `json:"email_address,omitempty"`
	OriginalReferralCode string `json:"original_referral_code,omitempty"`
	OriginalWaitlistCode string `json:"original_waitlist_code,omitempty"`
}

type PasswordReset struct {
	ID        int        `json:"id,omitempty"`
	Token     string     `json:"token,omitempty"`
	UserID    int        `json:"user_id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type PasswordChangeRequest struct {
	NewPassword     string `json:"new_password,omitempty"`
	CurrentPassword string `json:"current_password,omitempty"`
}

type ForgotPassword struct {
	EmailAddress string `json:"email_address,omitempty"`
}

type Account struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

type UserGoal struct {
	ID          int             `json:"id,omitempty"`
	UserID      int             `json:"user_id,omitempty"`
	Description string          `json:"description,omitempty"`
	Value       decimal.Decimal `json:"value,omitempty"`
	Rate        string          `json:"rate,omitempty"`
	Deadline    *NullTime       `json:"deadline,omitempty"`
	CompletedAt *NullTime       `json:"completed_at,omitempty"`
	IsActive    bool            `json:"is_active,omitempty"`
	CreatedAt   *NullTime       `json:"created_at,omitempty"`
	UpdatedAt   *NullTime       `json:"updated_at,omitempty"`
}

type GoalTemplate struct {
	ID           int       `json:"id,omitempty"`
	Name         string    `json:"name,omitempty"`
	Quantitative bool      `json:"quantitative,omitempty"`
	IsActive     bool      `json:"is_active,omitempty"`
	CreatedAt    *NullTime `json:"created_at,omitempty"`
	UpdatedAt    *NullTime `json:"updated_at,omitempty"`
}

type ModuleAccessAuthorizations struct {
	ID        int       `json:"id,omitempty"`
	ModuleID  int       `json:"module_id,omitempty"`
	UserID    int       `json:"user_id,omitempty"`
	CreatedAt *NullTime `json:"created_at,omitempty"`
	UpdatedAt *NullTime `json:"updated_at,omitempty"`
}

type QuizGrading struct {
	ID                int       `json:"id,omitempty"`
	ModuleID          int       `json:"module_id,omitempty"`
	QuizID            int       `json:"quiz_id,omitempty"`
	QuestionID        int       `json:"question_id,omitempty"`
	UserID            int       `json:"user_id,omitempty"`
	UserAnswerRanking int       `json:"user_answer_ranking,omitempty"`
	Correct           bool      `json:"correct,omitempty"`
	TakeNumber        int       `json:"take_number,omitempty"`
	CreatedAt         *NullTime `json:"created_at,omitempty"`
	UpdatedAt         *NullTime `json:"updated_at,omitempty"`
}

type LearningProgress struct {
	ID                int             `json:"id,omitempty"`
	UserID            sql.NullInt64   `json:"user_id,omitempty"`
	DeviceUniqueID    sql.NullString  `json:"device_unique_id,omitempty"`
	ModuleID          int             `json:"module_id,omitempty"`
	ModuleFileRanking int             `json:"module_file_ranking,omitempty"`
	Seek              decimal.Decimal `json:"seek,omitempty"`
	IsActive          bool            `json:"is_active,omitempty"`
	CreatedAt         *NullTime       `json:"created_at,omitempty"`
	UpdatedAt         *NullTime       `json:"updated_at,omitempty"`
}

type ProfileImage struct {
	ID        int       `json:"id,omitempty"`
	UserID    int       `json:"user_id,omitempty"`
	FileID    int       `json:"file_id,omitempty"`
	IsActive  bool      `json:"is_active,omitempty"`
	CreatedAt *NullTime `json:"created_at,omitempty"`
	UpdatedAt *NullTime `json:"updated_at,omitempty"`
	File      *File     `json:"file,omitempty"`
}

type Player struct {
	ID        int       `json:"id,omitempty"`
	UserID    int       `json:"user_id,omitempty"`
	Position  string    `json:"position,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Gender    string    `json:"gender,omitempty"`
	Weight    int       `json:"weight,omitempty"`
	Height    int       `json:"height,omitempty"`
	CreatedAt *NullTime `json:"created_at,omitempty"`
	UpdatedAt *NullTime `json:"updated_at,omitempty"`
}

type User struct {
	*Player        `json:"player,omitempty"`
	ID             int           `json:"id,omitempty"`
	About          string        `json:"about,omitempty"`
	DateOfBirth    *NullTime     `json:"date_of_birth,omitempty"`
	Email          string        `json:"email,omitempty"`
	Phone          int           `json:"phone,omitempty"`
	Location       string        `json:"location,omitempty"`
	Sports         string        `json:"sports,omitempty"`
	UserType       string        `json:"user_type,omitempty"`
	Hashtags       string        `json:"hashtags,omitempty"`
	IsActive       bool          `json:"is_active,omitempty"`
	LastOnline     *NullTime     `json:"last_online,omitempty"`
	UserAdminLevel string        `json:"user_admin_level,omitempty"`
	IsVerified     bool          `json:"is_verified,omitempty"`
	PasswordHash   string        `json:"password_hash,omitempty"`
	CreatedAt      *NullTime     `json:"created_at,omitempty"`
	UpdatedAt      *NullTime     `json:"updated_at,omitempty"`
	ProfileImage   *ProfileImage `json:"profile_image,omitempty"`

	LearningProgress           []*LearningProgress           `json:"learning_progress,omitempty"`
	QuizGradings               []*QuizGrading                `json:"quiz_gradings,omitempty"`
	ModuleAccessAuthorizations []*ModuleAccessAuthorizations `json:"moduleAccessAuthorizations,omitempty"`
	UserGoals                  []*UserGoal                   `json:"user_goals,omitempty"`
	CompletedModuleIDs         []int                         `json:"completed_module_ids,omitempty"`
	ReferralCode               *ReferralCode                 `json:"referral_code,omitempty"`
}

type NewUser struct {
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	ReferralCode string `json:"referral_code,omitempty"`
}

func (n *NewUser) IsValid() (bool, error) {
	if n.Email == "" {
		return false, fmt.Errorf("email")
	}
	if n.Password == "" {
		return false, fmt.Errorf("password")
	}
	if n.FirstName == "" {
		return false, fmt.Errorf("first_name")
	}
	if n.LastName == "" {
		return false, fmt.Errorf("last_name")
	}

	return true, nil
}
