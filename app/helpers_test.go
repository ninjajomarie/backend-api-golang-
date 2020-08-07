package external_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	external "github.com/johankaito/api.external/app"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-test/deep"
)

func (f *Fixture) Serve(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func (f *Fixture) ExpectStatus(rr *httptest.ResponseRecorder, wantStatus int) {
	if rr.Code != wantStatus {
		f.T.Errorf("status: Got=%v Want=%v Body=%v Func=%v", rr.Code, wantStatus, rr.Body, funcName(2))
	}
}

func (f *Fixture) ExpectAuthHeaders(rr *httptest.ResponseRecorder) {
	if strings.Join(rr.HeaderMap[external.AccessTokenHeader], "") == "" {
		f.T.Fatal("missing access token header")
	}
	if strings.Join(rr.HeaderMap[external.RefreshTokenHeader], "") == "" {
		f.T.Fatal("missing refresh token header")
	}
}

func (f *Fixture) ExpectNoAuthHeaders(rr *httptest.ResponseRecorder) {
	if strings.Join(rr.HeaderMap[external.AccessTokenHeader], "") != "" {
		f.T.Fatal("missing access token header")
	}
	if strings.Join(rr.HeaderMap[external.RefreshTokenHeader], "") != "" {
		f.T.Fatal("missing refresh token header")
	}
}

func (f *Fixture) ExpectEmptyObject(rr *httptest.ResponseRecorder) {
	if got := rr.Body.String(); got != "{}" {
		f.T.Errorf("expect empty object: Got=%v", got)
	}
}

func (f *Fixture) ExpectBodyContains(rr *httptest.ResponseRecorder, substr string) {
	if got := rr.Body.String(); !strings.Contains(got, substr) {
		f.T.Errorf("expect body contains: SubStr=%v Got=%v", substr, got)
	}
}

func (f *Fixture) ExpectErrorContains(err error, substr string) {
	if got := err.Error(); !strings.Contains(got, substr) {
		f.T.Errorf("expect error contains: SubStr=%v Got=%v", substr, got)
	}
}

func (f *Fixture) Bind(rr *httptest.ResponseRecorder, obj interface{}) {
	var objmap map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&objmap); err != nil {
		f.T.Errorf("bind: %v", err)
	}

	msg, ok := objmap["message"]
	if !ok {
		f.T.Error("missing message")
	}

	if err := json.Unmarshal(msg, &obj); err != nil {
		f.T.Errorf("unmarshal: %v", err)
	}
}

func (f *Fixture) UnAuthedRequest(method, url, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v0.1"+url, strings.NewReader(body))
	return f.Serve(req)
}

func (f *Fixture) GetAuthToken(email string) Auth {
	password := "keto"
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	f.ExpectNoError(err)
	f.InsertUser(external.NewUser{
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
	})

	rr := f.UnAuthedRequest(
		http.MethodPost,
		"/user/login",
		fmt.Sprintf(`{
			"email": "%s",
			"password": "%s"
		}`, email, password),
	)
	f.ExpectStatus(rr, http.StatusOK)

	var user external.User
	f.Bind(rr, &user)

	// auth headers set
	f.ExpectAuthHeaders(rr)

	return Auth{
		UserID:       user.ID,
		AccessToken:  strings.Join(rr.HeaderMap[external.AccessTokenHeader], ""),
		RefreshToken: strings.Join(rr.HeaderMap[external.RefreshTokenHeader], ""),
	}
}

func (f *Fixture) AuthedRequest(method, url, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	return f.Serve(req)
}

func (f *Fixture) JSONMarshal(s interface{}) string {
	b, err := json.Marshal(s)
	if err != nil {
		f.T.Fatal("marshaling to json")
	}
	return string(b)
}

func (f *TestHelper) ExpectStatusCode(err error, code codes.Code) {
	s, ok := status.FromError(err)
	if !ok {
		f.T.Fatal("error is not a grpc status error")
	}

	if s.Code() != code {
		f.T.Errorf("expected %v but got %v", code, s.Code())
	}
}

func (f *TestHelper) ExpectErrorContains(err error, contains string) {
	if err == nil {
		f.T.Fatal("error is nil")
	}
	if !strings.Contains(err.Error(), contains) {
		f.T.Errorf("expected contain %s but was %s", contains, err.Error())
	}
}

func (f *TestHelper) ExpectNoError(err error) {
	if err != nil {
		f.T.Fatalf("error was supposed to be nil but was %v at func: %v", err, funcName(2))
	}
}

func (f *TestHelper) ExpectDeepEq(lhs, rhs interface{}, tags ...string) {
	deep.CompareUnexportedFields = true // deep.Equal does not compare unexported fields (by default)
	if diff := deep.Equal(lhs, rhs); diff != nil {
		f.T.Fatal(tags, diff)
	}
}

func (f *DAOFixture) TruncateTables(tables ...string) {
	for _, table := range tables {
		if _, err := f.DAO.DB.Exec("TRUNCATE " + table + " CASCADE"); err != nil {
			f.T.Fatal(err)
		}
	}
}

func (f *DAOFixture) TruncateSchema(schema string) {
	// Inspiration: https://stackoverflow.com/questions/2829158/truncating-all-tables-in-a-postgres-database
	if _, err := f.DAO.DB.Exec(
		`
			CREATE OR REPLACE FUNCTION truncate_tables(schema IN VARCHAR) RETURNS void AS $$
			DECLARE
					statements CURSOR FOR
							SELECT tablename
							FROM pg_tables
							WHERE schemaname = schema;
					isNotRunning BOOLEAN;
					currentQuery TEXT;
			BEGIN
					FOR stmt IN statements LOOP
							SELECT FORMAT('TRUNCATE TABLE %I.%I CASCADE', schema, stmt.tablename)
							INTO currentQuery;

							SELECT COUNT(*) = 0
							INTO isNotRunning
							FROM pg_stat_activity
							WHERE query = currentQuery;

							IF isNotRunning THEN
								EXECUTE currentQuery;
							END IF;
					END LOOP;
			END;
			$$ LANGUAGE plpgsql;

			SELECT truncate_tables('` + schema + `')
		`,
	); err != nil {
		f.T.Fatal(err)
	}
}

func (f *DAOFixture) ExpectRowCount(tableName string, expectedCount int) {
	var count int
	err := f.DAO.DB.Get(&count, `
		SELECT count(*)
		FROM `+tableName,
	)
	if err != nil {
		f.T.Fatalf("error getting count results: %v", err)
	}
	if count != expectedCount {
		f.T.Errorf("%v expected count of %d but got %d", tableName, expectedCount, count)
	}
}

func (f *DAOFixture) ExpectRowCountWhere(tableName, where string, expectedCount int) {
	var count int
	err := f.DAO.DB.Get(&count, `
		SELECT count(*)
		FROM `+tableName+`
		WHERE `+where,
	)
	if err != nil {
		f.T.Fatalf("error getting count where results: %v", err)
	}
	if count != expectedCount {
		f.T.Errorf("%v expected count of %d but got %d", tableName, expectedCount, count)
	}
}

func (f *DAOFixture) ExpectRowCountWithJoinWhere(tableName, join, where string, expectedCount int) {
	var count int
	err := f.DAO.DB.Get(&count, `
		SELECT count(*)
		FROM `+tableName+`
		`+join+`
		WHERE `+where,
	)
	if err != nil {
		f.T.Fatalf("error getting count where results: %v", err)
	}
	if count != expectedCount {
		f.T.Errorf("%v expected count of %d but got %d", tableName, expectedCount, count)
	}
}

func (f *DAOFixture) ExpectRowCountMatchingStruct(
	tableName string,
	s interface{},
	count int,
	tags ...string,
) {
	where := ""

	// inspect each struct field
	r := reflect.ValueOf(s)
	for i := 0; i < r.Type().NumField(); i++ {
		field := r.Type().Field(i)
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]

		// Get value
		name := field.Name

		// Ignore auto-generated proto fields
		if strings.Contains(name, "XXX_") {
			continue
		}
		value := reflect.Indirect(r).FieldByName(name).Interface()

		// Ignore fields with zero values (unset fields)
		if !isZeroOfUnderlyingType(value) {
			if len(where) != 0 {
				where += "AND "
			}
			where += fmt.Sprintf("%v = :%v\n", jsonTag, jsonTag)
		}
	}

	rows, err := f.DAO.DB.NamedQuery(
		`
				SELECT count(*)
				FROM `+tableName+`
				WHERE `+where+`
		`,
		s,
	)
	f.ExpectNoError(err)

	var num int
	for rows.Next() {
		f.ExpectNoError(rows.Scan(&num))
	}

	f.ExpectDeepEq(num, count, tags...)
}

func isZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

func (f *DAOFixture) Close() {
	f.DAO.DB.Close()
	f.DAO.ReadDB.Close()
}

func (f *DAOFixture) MustTimestampProto(t time.Time) *timestamp.Timestamp {
	ts, err := ptypes.TimestampProto(t)
	f.ExpectNoError(err)
	// This is a little bit of a hack.
	// Because nanos aren't taken into account on insert our lives are much
	// easier if we cut them from the get-go.
	ts.Nanos = 0
	return ts
}

func (f *DAOFixture) MustTime(ts *timestamp.Timestamp) time.Time {
	t, err := ptypes.Timestamp(ts)
	f.ExpectNoError(err)
	return t
}

func (f *TestHelper) SetupEnv(config map[string]string) {
	os.Setenv("IS_KUBE", "true")
	for key, val := range config {
		f.MaybeSetEnv(key, val)
	}
}

func (f *TestHelper) MaybeSetEnv(key, val string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, val)
	}
}

func (f *DAOFixture) DumpTable(schema, table string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := "SELECT * FROM " + schema + "." + table
	rows, err := f.DAO.DB.QueryContext(ctx, query)
	if err != nil {
		f.T.Fatalf("querying rows: %v", err)
	}

	var objects []map[string]interface{}
	for rows.Next() {
		columns, err := rows.ColumnTypes()
		if err != nil {
			f.T.Fatalf("getting column types from rows: %v", err)
		}

		values := make([]interface{}, len(columns))
		object := map[string]interface{}{}
		for i, column := range columns {
			object[column.Name()] = reflect.New(column.ScanType()).Interface()
			values[i] = object[column.Name()]
		}

		err = rows.Scan(values...)
		if err != nil {
			f.T.Fatalf("scanning rows values: %v", err)
		}

		objects = append(objects, object)
		bytes, err := json.MarshalIndent(objects, "", "\t")
		if err != nil {
			f.T.Fatalf("marshaling: %v", err)
		}
		fmt.Println(string(bytes))
	}
}

// ExpectStructMatch will compare the actual to expected struct.
// More specifically it will check that all non-zero fields
// in `expect` match their counterpart in `actual`.
// This allows you to make partial assertions re. the contents of a struct.
func (f *TestHelper) ExpectStructMatch(actual, expect interface{}) {
	vA := reflect.ValueOf(actual)
	vE := reflect.ValueOf(expect)

	var differences []string
	for i := 0; i < vE.Type().NumField(); i++ {
		fieldE := vE.Type().Field(i)
		nameE := fieldE.Name

		// Ignore auto-generated proto fields
		if strings.Contains(nameE, "XXX_") {
			continue
		}

		valueE := reflect.Indirect(vE).FieldByName(nameE).Interface()

		// Ignore fields with zero values (unset fields)
		if isZeroOfUnderlyingType(valueE) {
			continue
		}

		valueA := reflect.Indirect(vA).FieldByName(nameE).Interface()

		diffs := deep.Equal(valueE, valueA)
		for _, d := range diffs {
			differences = append(differences, nameE+": "+d)
		}
	}

	if len(differences) != 0 {
		f.T.Errorf("Struct mismatch: %q", differences)
	}
}

// FuncName gives the function name of the caller.
func funcName(n int) string {
	pc, _, _, _ := runtime.Caller(n)
	name := runtime.FuncForPC(pc).Name()
	nameParts := strings.Split(name, ".")
	return nameParts[len(nameParts)-1]
}
