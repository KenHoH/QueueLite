package app

import (
	"context"
	"errors"
	"strings"

	"QueueLite/internal/apperror"
	"QueueLite/internal/business/domain"
	counterapp "QueueLite/internal/counter/app"
	counterdomain "QueueLite/internal/counter/domain"
	subscriptionapp "QueueLite/internal/subscription/app"
	"QueueLite/internal/validation"

	"github.com/google/uuid"
)

type BusinessService struct {
	repo                BusinessRepo
	subscriptionService *subscriptionapp.SubscriptionService
	counterService      *counterapp.CounterService
}

func NewBusinessService(repo BusinessRepo, subscriptionService *subscriptionapp.SubscriptionService, counterService ...*counterapp.CounterService) *BusinessService {
	service := &BusinessService{repo: repo, subscriptionService: subscriptionService}
	if len(counterService) > 0 {
		service.counterService = counterService[0]
	}
	return service
}

func (s *BusinessService) CreateBusiness(ctx context.Context, business domain.Business) (*domain.Business, error) {
	business.Name = strings.TrimSpace(business.Name)
	business.Location = strings.TrimSpace(business.Location)
	business.Email = strings.TrimSpace(business.Email)
	business.PhoneNumber = strings.TrimSpace(business.PhoneNumber)

	if business.Name == "" {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_NAME_REQUIRED", "business name is required")
	}
	if business.Location == "" {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_LOCATION_REQUIRED", "business location is required")
	}
	if business.OpenTime.IsZero() || business.CloseTime.IsZero() {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_OPERATIONAL_TIME_REQUIRED", "business open and close time are required")
	}
	if business.Email == "" {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_EMAIL_REQUIRED", "business email is required")
	}
	if business.PhoneNumber == "" {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_PHONE_NUMBER_REQUIRED", "business phone number is required")
	}
	if !validation.Email(business.Email) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_EMAIL", "invalid email address")
	}
	if !validation.Phone(business.PhoneNumber) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_PHONE_NUMBER", "invalid phone number")
	}

	record, err := s.repo.CreateBusiness(ctx, &business)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CREATE_BUSINESS_ERROR", "failed to create business", err)
	}
	if s.subscriptionService != nil {
		if _, err := s.subscriptionService.CreateDefaultBusinessSubscriptionPlan(ctx, record.ID); err != nil {
			return nil, err
		}
	}
	return record, nil
}

func (s *BusinessService) RegisterBusiness(ctx context.Context, ownerUserID uuid.UUID, business domain.Business) (*domain.Business, error) {
	record, err := s.CreateBusiness(ctx, business)
	if err != nil {
		return nil, err
	}
	if ownerUserID != uuid.Nil {
		if err := s.repo.CreateUserBusinessRelation(ctx, record.ID, ownerUserID, "owner"); err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "CREATE_BUSINESS_OWNER_ERROR", "failed to create business owner relation", err)
		}
	}
	if s.counterService != nil {
		if _, err := s.counterService.CreateCounter(ctx, counterdomain.Counter{BusinessID: record.ID, Name: "Default Counter"}); err != nil {
			return nil, err
		}
	}
	return record, nil
}

func (s *BusinessService) GetBusiness(ctx context.Context, id uuid.UUID) (*domain.Business, error) {
	business, err := s.repo.GetBusiness(ctx, id)
	if err != nil {
		if errors.Is(err, ErrBusinessNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "BUSINESS_NOT_FOUND", "business not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_ERROR", "failed to get business", err)
	}
	return business, nil
}

func (s *BusinessService) UpdateBusiness(ctx context.Context, businessID uuid.UUID, business domain.UpdateBusiness) error {
	if business.Name != nil {
		name := strings.TrimSpace(*business.Name)
		if name == "" {
			return apperror.New(apperror.KindInvalid, "BUSINESS_NAME_REQUIRED", "business name is required")
		}
		business.Name = &name
	}
	if business.Location != nil {
		location := strings.TrimSpace(*business.Location)
		if location == "" {
			return apperror.New(apperror.KindInvalid, "BUSINESS_LOCATION_REQUIRED", "business location is required")
		}
		business.Location = &location
	}
	if business.Description != nil {
		description := strings.TrimSpace(*business.Description)
		business.Description = &description
	}
	if business.Email != nil {
		email := strings.TrimSpace(*business.Email)
		if email == "" {
			return apperror.New(apperror.KindInvalid, "BUSINESS_EMAIL_REQUIRED", "business email is required")
		}
		business.Email = &email
		if !validation.Email(email) {
			return apperror.New(apperror.KindInvalid, "INVALID_EMAIL", "invalid email address")
		}
	}
	if business.PhoneNumber != nil {
		phoneNumber := strings.TrimSpace(*business.PhoneNumber)
		if phoneNumber == "" {
			return apperror.New(apperror.KindInvalid, "BUSINESS_PHONE_NUMBER_REQUIRED", "business phone number is required")
		}
		business.PhoneNumber = &phoneNumber
		if !validation.Phone(phoneNumber) {
			return apperror.New(apperror.KindInvalid, "INVALID_PHONE_NUMBER", "invalid phone number")
		}
	}
	if (business.OpenTime != nil && business.OpenTime.IsZero()) || (business.CloseTime != nil && business.CloseTime.IsZero()) {
		return apperror.New(apperror.KindInvalid, "BUSINESS_OPERATIONAL_TIME_REQUIRED", "business open and close time are required")
	}

	if err := s.repo.UpdateBusiness(ctx, businessID, &business); err != nil {
		if errors.Is(err, ErrBusinessNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "BUSINESS_NOT_FOUND", "business not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_BUSINESS_ERROR", "failed to update business", err)
	}
	return nil
}

func (s *BusinessService) DeleteBusiness(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteBusiness(ctx, id); err != nil {
		if errors.Is(err, ErrBusinessNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "BUSINESS_NOT_FOUND", "business not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_BUSINESS_ERROR", "failed to delete business", err)
	}
	return nil
}

func (s *BusinessService) SearchBusiness(ctx context.Context, searchQuery string) ([]domain.Business, error) {
	businesses, err := s.repo.SearchBusiness(ctx, strings.TrimSpace(searchQuery))
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "SEARCH_FAILURE", "search failed", err)
	}
	return businesses, nil
}

func (s *BusinessService) GetAllBusiness(ctx context.Context, cursor *domain.BusinessCursor, limit int) ([]domain.Business, *domain.BusinessCursor, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	businesses, nextCursor, err := s.repo.GetAllBusiness(ctx, cursor, limit)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.KindInternal, "GET_ALL_BUSINESS", "get all business pagination failed", err)
	}
	return businesses, nextCursor, nil
}

func (s *BusinessService) GetUserBusinesses(ctx context.Context, userID uuid.UUID) ([]domain.BusinessMembership, error) {
	items, err := s.repo.GetUserBusinesses(ctx, userID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "GET_MY_BUSINESSES_ERROR", "failed to get your businesses", err)
	}
	return items, nil
}

func (s *BusinessService) UpsertBusinessMember(ctx context.Context, actorID, businessID uuid.UUID, identifier, role string) (*domain.BusinessMember, error) {
	identifier = strings.TrimSpace(identifier)
	role = strings.ToLower(strings.TrimSpace(role))
	if identifier == "" {
		return nil, apperror.New(apperror.KindInvalid, "MEMBER_IDENTIFIER_REQUIRED", "username or email is required")
	}
	if role != "admin" && role != "counter" {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_MEMBER_ROLE", "role must be admin or counter")
	}
	actorRole, err := s.repo.GetUserBusinessRole(ctx, actorID, businessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check business access", err)
	}
	if actorRole != "owner" && actorRole != "admin" {
		return nil, apperror.New(apperror.KindForbidden, "BUSINESS_ACCESS_DENIED", "you cannot manage this business")
	}
	if actorRole == "admin" && role != "counter" {
		return nil, apperror.New(apperror.KindForbidden, "MEMBER_ROLE_DENIED", "admins may only assign the counter role")
	}
	membershipRepo, ok := s.repo.(MembershipAdminRepo)
	if !ok {
		return nil, apperror.New(apperror.KindInternal, "MEMBER_ADMIN_UNAVAILABLE", "business member administration is unavailable")
	}
	user, err := membershipRepo.FindMembershipUser(ctx, identifier, strings.Contains(identifier, "@"))
	if err != nil {
		if errors.Is(err, domain.ErrMembershipUserNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "USER_NOT_FOUND", "no account matches that username or email", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "MEMBER_LOOKUP_ERROR", "failed to find account", err)
	}
	if user.ID == actorID {
		return nil, apperror.New(apperror.KindForbidden, "MEMBER_SELF_CHANGE", "you cannot change your own business role")
	}
	existingRole, err := s.repo.GetUserBusinessRole(ctx, user.ID, businessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check member role", err)
	}
	if existingRole == "owner" {
		return nil, apperror.New(apperror.KindForbidden, "BUSINESS_OWNER_PROTECTED", "the owner role cannot be changed")
	}
	if actorRole == "admin" && existingRole != "" && existingRole != "counter" {
		return nil, apperror.New(apperror.KindForbidden, "MEMBER_ROLE_DENIED", "admins cannot change another administrator")
	}
	if err := membershipRepo.UpsertUserBusinessRelation(ctx, businessID, user.ID, role); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "UPSERT_BUSINESS_MEMBER_ERROR", "failed to save business member", err)
	}
	return &domain.BusinessMember{UserID: user.ID, Username: user.Username, Role: role}, nil
}

func (s *BusinessService) RequireManager(ctx context.Context, userID, businessID uuid.UUID) error {
	role, err := s.repo.GetUserBusinessRole(ctx, userID, businessID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "BUSINESS_ACCESS_ERROR", "failed to check business access", err)
	}
	if role != "owner" && role != "admin" {
		return apperror.New(apperror.KindForbidden, "BUSINESS_ACCESS_DENIED", "you cannot manage this business")
	}
	return nil
}
