package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/business/domain"
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type BusinessService struct {
	repo BusinessRepo
}

func NewBusinessService(repo BusinessRepo) *BusinessService {
	return &BusinessService{repo: repo}
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

	record, err := s.repo.CreateBusiness(ctx, &business)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CREATE_BUSINESS_ERROR", "failed to create business", err)
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
	}
	if business.PhoneNumber != nil {
		phoneNumber := strings.TrimSpace(*business.PhoneNumber)
		if phoneNumber == "" {
			return apperror.New(apperror.KindInvalid, "BUSINESS_PHONE_NUMBER_REQUIRED", "business phone number is required")
		}
		business.PhoneNumber = &phoneNumber
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
