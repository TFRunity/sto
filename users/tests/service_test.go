package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AlexeBabanov/sto/users/internal"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Общее: тестовые данные
// ---------------------------------------------------------------------------

const (
	testSecret = "test-secret"
	testTTL    = time.Hour

	clientPhone      = "+79990000001" // уже в «базе», роль client
	clientPassword   = "client-password"
	engineerPhone    = "+70000000001" // уже в «базе», роль engineer
	engineerPassword = "engineer-password"
	newPhone         = "+79990000002" // свободный телефон
)

// ---------------------------------------------------------------------------
// Общее: fake репозитория (ADR-013) — пользователи в памяти, ключ — телефон
// ---------------------------------------------------------------------------

type fakeStore struct {
	byPhone map[string]internal.User
}

func (s *fakeStore) CreateClient(_ context.Context, u internal.User) (internal.User, error) {
	if _, taken := s.byPhone[u.Phone]; taken {
		return internal.User{}, internal.ErrPhoneTaken
	}
	u.Role = internal.RoleClient // как в SQL: роль ставит запрос, а не сервис
	s.byPhone[u.Phone] = u
	return u, nil
}

func (s *fakeStore) GetByPhone(_ context.Context, phone string) (internal.User, error) {
	u, ok := s.byPhone[phone]
	if !ok {
		return internal.User{}, internal.ErrUserNotFound
	}
	return u, nil
}

// ---------------------------------------------------------------------------
// Общее: подготовка — сервис и «база» с клиентом и инженером
// ---------------------------------------------------------------------------

type fixture struct {
	svc      *internal.Service
	store    *fakeStore
	client   internal.User
	engineer internal.User
}

func setup(t *testing.T) fixture {
	t.Helper()

	client := existingUser(t, clientPhone, clientPassword, internal.RoleClient)
	engineer := existingUser(t, engineerPhone, engineerPassword, internal.RoleEngineer)
	store := &fakeStore{byPhone: map[string]internal.User{
		client.Phone:   client,
		engineer.Phone: engineer,
	}}

	svc, err := internal.NewService(store, []byte(testSecret), testTTL)
	require.NoError(t, err)

	return fixture{svc: svc, store: store, client: client, engineer: engineer}
}

// MinCost: тесту нужен валидный хеш, а не стойкий.
func existingUser(t *testing.T, phone, password string, role internal.Role) internal.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return internal.User{ID: uuid.New(), Name: "Тест", Phone: phone, PasswordHash: string(hash), Role: role}
}

// ---------------------------------------------------------------------------
// Общее: проверки
// ---------------------------------------------------------------------------

// checkToken разбирает токен так же, как это будет делать Gateway (ADR-023).
func checkToken(t *testing.T, token string, wantID uuid.UUID, wantRole internal.Role) {
	t.Helper()

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return []byte(testSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	require.NoError(t, err)

	sub, err := claims.GetSubject()
	require.NoError(t, err)
	assert.Equal(t, wantID.String(), sub)

	assert.Equal(t, string(wantRole), claims["role"])

	exp, err := claims.GetExpirationTime()
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.WithinDuration(t, time.Now().Add(testTTL), exp.Time, time.Minute)
}

// ---------------------------------------------------------------------------
// Use case: регистрация клиента
// ---------------------------------------------------------------------------

func TestRegister(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		password string
		wantErr  error
	}{
		{name: "успех", phone: newPhone, password: "correct-password"},
		{name: "пароль ровно 72 байта кириллицей", phone: newPhone, password: strings.Repeat("я", 36)},
		{name: "пароль 73 байта кириллицей", phone: newPhone, password: strings.Repeat("я", 36) + "a", wantErr: internal.ErrPasswordTooLong},
		{name: "телефон занят", phone: clientPhone, password: "correct-password", wantErr: internal.ErrPhoneTaken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setup(t) // своя «база» на кейс: регистрация её меняет

			id, token, err := f.svc.Register(t.Context(), "Иван", tt.phone, tt.password)

			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				return
			}
			saved := f.store.byPhone[tt.phone]
			assert.Equal(t, id, saved.ID)
			assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(saved.PasswordHash), []byte(tt.password)),
				"сохранённый хеш не подходит к паролю")
			checkToken(t, token, id, internal.RoleClient)
		})
	}
}

// ---------------------------------------------------------------------------
// Use case: вход
// ---------------------------------------------------------------------------

func TestLogin(t *testing.T) {
	f := setup(t) // одна «база» на все кейсы: вход её только читает

	tests := []struct {
		name     string
		phone    string
		password string
		wantUser internal.User
		wantErr  error
	}{
		{name: "успех, клиент", phone: clientPhone, password: clientPassword, wantUser: f.client},
		{name: "успех, роль берётся из базы", phone: engineerPhone, password: engineerPassword, wantUser: f.engineer},
		{name: "неверный пароль", phone: clientPhone, password: "wrong-password", wantErr: internal.ErrInvalidCredentials},
		{name: "неизвестный телефон", phone: newPhone, password: clientPassword, wantErr: internal.ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := f.svc.Login(t.Context(), tt.phone, tt.password)

			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				return
			}
			checkToken(t, token, tt.wantUser.ID, tt.wantUser.Role)
		})
	}
}
