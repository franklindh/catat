package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/franklindh/catat/db/sqlc"
	"github.com/franklindh/catat/token"
	"github.com/franklindh/catat/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type dashboardSummaryResponse struct {
	TotalIncome  pgtype.Numeric `json:"total_income"`
	TotalExpense pgtype.Numeric `json:"total_expense"`
}

type expenseByCategoryResponse struct {
	CategoryName     string         `json:"category_name"`
	IconUrl          string         `json:"icon_url"`
	TotalAmount      pgtype.Numeric `json:"total_amount"`
	TransactionCount int64          `json:"transaction_count"`
}

type dailyExpenseTrendResponse struct {
	Date        string         `json:"date"`
	TotalAmount pgtype.Numeric `json:"total_amount"`
}

type totalBalanceResponse struct {
	CurrentBalance pgtype.Numeric `json:"current_balance"`
}

func parseDateQuery(ctx *gin.Context, paramName string, defaultValue time.Time) (pgtype.Timestamptz, error) {
	dateStr := ctx.Query(paramName)
	if dateStr == "" {
		return pgtype.Timestamptz{Time: defaultValue, Valid: true}, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {

		parsedTime, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return pgtype.Timestamptz{}, err
		}
	}

	return pgtype.Timestamptz{Time: parsedTime, Valid: true}, nil
}

func (s *Server) getDashboard(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	now := time.Now()
	startDate, err := parseDateQuery(ctx, "start_date", now.AddDate(0, 0, -30))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format")))
		return
	}

	endDate, err := parseDateQuery(ctx, "end_date", now)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format")))
		return
	}

	dashboard, err := s.store.GetDashboardSummary(ctx, db.GetDashboardSummaryParams{
		UserID:            authPayload.UserID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := dashboardSummaryResponse{
		TotalIncome:  dashboard.TotalIncome,
		TotalExpense: dashboard.TotalExpense,
	}
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) getTotalBalance(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	balance, err := s.store.GetTotalBalance(ctx, authPayload.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	rsp := totalBalanceResponse{
		CurrentBalance: balance,
	}
	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) getExpenseByCategory(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	now := time.Now()
	startDate, err := parseDateQuery(ctx, "start_date", now.AddDate(0, 0, -30))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format")))
		return
	}

	endDate, err := parseDateQuery(ctx, "end_date", now)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format")))
		return
	}

	expenses, err := s.store.GetExpenseByCategory(ctx, db.GetExpenseByCategoryParams{
		UserID:            authPayload.UserID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	var rsp []expenseByCategoryResponse
	for _, expense := range expenses {
		iconUrl := ""
		if expense.IconUrl.Valid {
			iconUrl = expense.IconUrl.String
		}
		rsp = append(rsp, expenseByCategoryResponse{
			CategoryName:     expense.CategoryName,
			IconUrl:          iconUrl,
			TotalAmount:      expense.TotalAmount,
			TransactionCount: expense.TransactionCount,
		})
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (s *Server) getDailyExpenseTrend(ctx *gin.Context) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("authorization required")))
		return
	}

	authPayload, ok := payload.(*token.Payload)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, util.ErrorResponse(errors.New("invalid authorization payload")))
		return
	}

	now := time.Now()
	startDate, err := parseDateQuery(ctx, "start_date", now.AddDate(0, 0, -30))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format")))
		return
	}

	endDate, err := parseDateQuery(ctx, "end_date", now)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.ErrorResponse(errors.New("invalid date format")))
		return
	}

	trends, err := s.store.GetDailyExpenseTrend(ctx, db.GetDailyExpenseTrendParams{
		UserID:            authPayload.UserID,
		TransactionDate:   startDate,
		TransactionDate_2: endDate,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, util.ErrorResponse(err))
		return
	}

	var rsp []dailyExpenseTrendResponse
	for _, trend := range trends {
		dateStr := ""
		if trend.Date.Valid {
			dateStr = trend.Date.Time.Format("2006-01-02")
		}
		rsp = append(rsp, dailyExpenseTrendResponse{
			Date:        dateStr,
			TotalAmount: trend.TotalAmount,
		})
	}

	ctx.JSON(http.StatusOK, rsp)
}
