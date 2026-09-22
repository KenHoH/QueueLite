package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/business/domain"
)

func toBusinessRecord(business *domain.Business) model.Business {
	return model.Business{
		ID:          business.ID,
		Name:        business.Name,
		Location:    business.Location,
		Description: business.Description,
		Operational: business.Operational,
		OpenTime:    business.OpenTime,
		CloseTime:   business.CloseTime,
		Email:       business.Email,
		PhoneNumber: business.PhoneNumber,
		CreatedAt:   business.CreatedAt,
	}
}

func toDomainBusiness(record *model.Business) *domain.Business {
	return &domain.Business{
		ID:          record.ID,
		Name:        record.Name,
		Location:    record.Location,
		Description: record.Description,
		Operational: record.Operational,
		OpenTime:    record.OpenTime,
		CloseTime:   record.CloseTime,
		CreatedAt:   record.CreatedAt,
		Email:       record.Email,
		PhoneNumber: record.PhoneNumber,
	}
}

func toDomainBusinesses(records []model.Business) []domain.Business {
	businesses := make([]domain.Business, 0, len(records))
	for i := range records {
		businesses = append(businesses, *toDomainBusiness(&records[i]))
	}
	return businesses
}
