package app

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"subtrackr/internal/config"
	"subtrackr/internal/database"
	"subtrackr/internal/handlers"
	"subtrackr/internal/middleware"
	"subtrackr/internal/repository"
	"subtrackr/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

const reminderDispatchHour = 10

func Run() {
	flag.Parse()

	cfg := config.Load()

	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	subscriptionRepo := repository.NewSubscriptionRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	exchangeRateRepo := repository.NewExchangeRateRepository(db)
	userRepo := repository.NewUserRepository(db)

	categoryService := service.NewCategoryService(categoryRepo)
	currencyService := service.NewCurrencyService(exchangeRateRepo)
	subscriptionService := service.NewSubscriptionService(subscriptionRepo, categoryService)
	settingsService := service.NewSettingsService(settingsRepo)
	userService := service.NewUserService(userRepo)
	telegramService := service.NewTelegramService(settingsService.ForUser(0))
	webhookService := service.NewWebhookService(settingsService.ForUser(0))
	logoService := service.NewLogoService()

	sessionSecret, err := settingsService.GetOrGenerateSessionSecret()
	if err != nil {
		log.Fatal("Failed to initialize session secret:", err)
	}
	sessionService := service.NewSessionService(sessionSecret)

	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionService, settingsService, currencyService, telegramService, webhookService, logoService, cfg.DevPreview)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	authHandler := handlers.NewAuthHandler(userService, settingsService, sessionService)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.SetFuncMap(templateFuncMap())

	tmpl := loadTemplates()
	if tmpl != nil && len(tmpl.Templates()) > 0 {
		router.SetHTMLTemplate(tmpl)
	} else {
		router.LoadHTMLGlob("templates/*")
	}

	router.Static("/static", "./web/static")
	router.StaticFile("/favicon.ico", "./web/static/favicon.ico")
	router.StaticFile("/manifest.json", "./web/static/manifest.json")

	router.GET("/healthz", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database connection unavailable"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database ping failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.Use(middleware.AuthMiddleware(userService, sessionService))
	setupRoutes(router, subscriptionHandler, settingsHandler, settingsService, categoryHandler, authHandler)

	go startRenewalReminderScheduler(userService, subscriptionService, settingsService)
	go startCancellationReminderScheduler(userService, subscriptionService, settingsService)

	port := cfg.Port
	log.Printf("SubTrackr server starting on port %s", port)
	log.Printf("Server running at http://localhost:%s", port)
	log.Fatal(router.Run(":" + port))
}

func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"int": func(v interface{}) int {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case time.Month:
				return int(val)
			default:
				return 0
			}
		},
		"fmtDate": func(t *time.Time, format string) string {
			if t == nil {
				return ""
			}
			return t.Format(format)
		},
		"fmtTime": func(t time.Time, format string) string {
			return t.Format(format)
		},
	}
}

func loadTemplates() *template.Template {
	tmpl := template.New("")
	tmpl.Funcs(templateFuncMap())

	criticalTemplates := []string{
		"templates/dashboard.html",
		"templates/subscriptions.html",
		"templates/error.html",
	}

	templateFiles := []string{
		"templates/dashboard.html",
		"templates/subscriptions.html",
		"templates/analytics.html",
		"templates/calendar.html",
		"templates/settings.html",
		"templates/subscription-form.html",
		"templates/subscription-list.html",
		"templates/categories-list.html",
		"templates/api-keys-list.html",
		"templates/settings-message.html",
		"templates/form-errors.html",
		"templates/error.html",
		"templates/login.html",
		"templates/register.html",
		"templates/login-error.html",
	}

	var parsedCount int
	var failedCount int
	var missingCritical []string

	for _, file := range templateFiles {
		if _, err := os.Stat(file); err != nil {
			log.Printf("Warning: Template file not found: %s", file)
			for _, critical := range criticalTemplates {
				if critical == file {
					missingCritical = append(missingCritical, file)
				}
			}
			continue
		}

		if _, err := tmpl.ParseFiles(file); err != nil {
			log.Printf("Error: Failed to parse template %s: %v", file, err)
			failedCount++
			for _, critical := range criticalTemplates {
				if critical == file {
					missingCritical = append(missingCritical, file)
				}
			}
		} else {
			parsedCount++
		}
	}

	log.Printf("Template loading summary: %d parsed, %d failed, %d total", parsedCount, failedCount, len(templateFiles))

	if len(missingCritical) > 0 {
		log.Fatalf("Critical templates failed to load: %v. Application cannot continue.", missingCritical)
	}

	if failedCount > len(templateFiles)/2 {
		log.Printf("Warning: More than half of templates failed to load (%d/%d). Application may not function correctly.", failedCount, len(templateFiles))
	}

	return tmpl
}

func setupRoutes(router *gin.Engine, handler *handlers.SubscriptionHandler, settingsHandler *handlers.SettingsHandler, settingsService *service.SettingsService, categoryHandler *handlers.CategoryHandler, authHandler *handlers.AuthHandler) {
	router.GET("/register", authHandler.ShowRegisterPage)
	router.GET("/login", authHandler.ShowLoginPage)

	router.GET("/", handler.Dashboard)
	router.GET("/dashboard", handler.Dashboard)
	router.GET("/subscriptions", handler.SubscriptionsList)
	router.GET("/analytics", handler.Analytics)
	router.GET("/calendar", handler.Calendar)
	router.GET("/settings", handler.Settings)

	form := router.Group("/form")
	{
		form.GET("/subscription", handler.GetSubscriptionForm)
		form.GET("/subscription/:id", handler.GetSubscriptionForm)
	}

	api := router.Group("/api")
	{
		api.GET("/subscriptions", handler.GetSubscriptions)
		api.POST("/subscriptions", handler.CreateSubscription)
		api.GET("/subscriptions/:id", handler.GetSubscription)
		api.PUT("/subscriptions/:id", handler.UpdateSubscription)
		api.DELETE("/subscriptions/:id", handler.DeleteSubscription)
		api.GET("/stats", handler.GetStats)

		api.GET("/export/csv", handler.ExportCSV)
		api.GET("/export/json", handler.ExportJSON)

		api.POST("/settings/telegram", settingsHandler.SaveTelegramSettings)
		api.POST("/settings/telegram/test", settingsHandler.TestTelegramConnection)
		api.GET("/settings/telegram", settingsHandler.GetTelegramConfig)
		api.POST("/settings/webhook", settingsHandler.SaveWebhookSettings)
		api.POST("/settings/webhook/test", settingsHandler.TestWebhookConnection)
		api.POST("/settings/notifications/:setting", settingsHandler.UpdateNotificationSetting)
		api.GET("/settings/notifications", settingsHandler.GetNotificationSettings)
		api.GET("/settings/apikeys", settingsHandler.ListAPIKeys)
		api.POST("/settings/apikeys", settingsHandler.CreateAPIKey)
		api.DELETE("/settings/apikeys/:id", settingsHandler.DeleteAPIKey)
		api.POST("/settings/currency", settingsHandler.UpdateCurrency)
		api.POST("/settings/date-format", settingsHandler.UpdateDateFormat)
		api.POST("/settings/dark-mode", settingsHandler.ToggleDarkMode)
		api.GET("/categories", categoryHandler.ListCategories)
		api.POST("/categories", categoryHandler.CreateCategory)
		api.PUT("/categories/:id", categoryHandler.UpdateCategory)
		api.DELETE("/categories/:id", categoryHandler.DeleteCategory)
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.GET("/auth/logout", authHandler.Logout)
		api.GET("/settings/theme", settingsHandler.GetTheme)
		api.POST("/settings/theme", settingsHandler.SetTheme)
		api.GET("/settings/personalization", settingsHandler.GetUIPersonalizationSettings)
		api.POST("/settings/personalization", settingsHandler.SaveUIPersonalizationSettings)
	}

	v1 := router.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(settingsService))
	{
		v1.GET("/subscriptions", handler.GetSubscriptionsAPI)
		v1.POST("/subscriptions", handler.CreateSubscription)
		v1.GET("/subscriptions/:id", handler.GetSubscription)
		v1.PUT("/subscriptions/:id", handler.UpdateSubscription)
		v1.DELETE("/subscriptions/:id", handler.DeleteSubscription)
		v1.GET("/stats", handler.GetStats)
		v1.GET("/export/csv", handler.ExportCSV)
		v1.GET("/export/json", handler.ExportJSON)
	}
}

func startRenewalReminderScheduler(userService *service.UserService, subscriptionService *service.SubscriptionService, settingsService *service.SettingsService) {
	go func() {
		time.Sleep(15 * time.Second)
		checkAndSendRenewalReminders(userService, subscriptionService, settingsService)

		for {
			sleepFor := durationUntilNextReminderRun()
			log.Printf("Next renewal reminder scan scheduled in %s", sleepFor.Round(time.Second))
			time.Sleep(sleepFor)
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Panic in renewal reminder check: %v", r)
					}
				}()
				checkAndSendRenewalReminders(userService, subscriptionService, settingsService)
			}()
		}
	}()
}

func checkAndSendRenewalReminders(userService *service.UserService, subscriptionService *service.SubscriptionService, settingsService *service.SettingsService) {
	users, err := userService.ListAll()
	if err != nil {
		log.Printf("Error listing users for renewal reminders: %v", err)
		return
	}

	for _, user := range users {
		userSettings := settingsService.ForUser(user.ID)
		enabled, err := userSettings.GetBoolSetting("renewal_reminders", false)
		if err != nil || !enabled {
			continue
		}

		reminderDays := userSettings.GetIntSettingWithDefault("reminder_days", 7)
		if reminderDays <= 0 {
			continue
		}

		userSubscriptions := subscriptionService.ForUser(user.ID)
		subscriptions, err := userSubscriptions.GetSubscriptionsNeedingReminders(reminderDays)
		if err != nil {
			log.Printf("Error getting subscriptions for renewal reminders (user %d): %v", user.ID, err)
			continue
		}

		telegramService := service.NewTelegramService(userSettings)
		webhookService := service.NewWebhookService(userSettings)

		for sub, daysUntil := range subscriptions {
			telegramErr := telegramService.SendRenewalReminder(sub, daysUntil)
			webhookErr := webhookService.SendRenewalReminder(sub, daysUntil)
			if telegramErr != nil && webhookErr != nil {
				log.Printf("Error sending renewal reminder for user %d subscription %s (ID: %d): telegram=%v, webhook=%v", user.ID, sub.Name, sub.ID, telegramErr, webhookErr)
				continue
			}

			now := time.Now()
			sub.LastReminderSent = &now
			if sub.RenewalDate != nil {
				renewalDateCopy := *sub.RenewalDate
				sub.LastReminderRenewalDate = &renewalDateCopy
			}
			if _, updateErr := userSubscriptions.Update(sub.ID, sub); updateErr != nil {
				log.Printf("Warning: Failed to update last reminder sent for user %d subscription %s (ID: %d): %v", user.ID, sub.Name, sub.ID, updateErr)
			}
		}
	}
}

func startCancellationReminderScheduler(userService *service.UserService, subscriptionService *service.SubscriptionService, settingsService *service.SettingsService) {
	go func() {
		time.Sleep(15 * time.Second)
		checkAndSendCancellationReminders(userService, subscriptionService, settingsService)

		for {
			sleepFor := durationUntilNextReminderRun()
			log.Printf("Next cancellation reminder scan scheduled in %s", sleepFor.Round(time.Second))
			time.Sleep(sleepFor)
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Panic in cancellation reminder check: %v", r)
					}
				}()
				checkAndSendCancellationReminders(userService, subscriptionService, settingsService)
			}()
		}
	}()
}

func durationUntilNextReminderRun() time.Duration {
	now := time.Now().In(time.Local)
	next := time.Date(now.Year(), now.Month(), now.Day(), reminderDispatchHour, 0, 0, 0, time.Local)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func checkAndSendCancellationReminders(userService *service.UserService, subscriptionService *service.SubscriptionService, settingsService *service.SettingsService) {
	users, err := userService.ListAll()
	if err != nil {
		log.Printf("Error listing users for cancellation reminders: %v", err)
		return
	}

	for _, user := range users {
		userSettings := settingsService.ForUser(user.ID)
		enabled, err := userSettings.GetBoolSetting("cancellation_reminders", false)
		if err != nil || !enabled {
			continue
		}

		reminderDays := userSettings.GetIntSettingWithDefault("cancellation_reminder_days", 7)
		if reminderDays <= 0 {
			continue
		}

		userSubscriptions := subscriptionService.ForUser(user.ID)
		subscriptions, err := userSubscriptions.GetSubscriptionsNeedingCancellationReminders(reminderDays)
		if err != nil {
			log.Printf("Error getting subscriptions for cancellation reminders (user %d): %v", user.ID, err)
			continue
		}

		telegramService := service.NewTelegramService(userSettings)
		webhookService := service.NewWebhookService(userSettings)

		for sub, daysUntil := range subscriptions {
			telegramErr := telegramService.SendCancellationReminder(sub, daysUntil)
			webhookErr := webhookService.SendCancellationReminder(sub, daysUntil)
			if telegramErr != nil && webhookErr != nil {
				log.Printf("Error sending cancellation reminder for user %d subscription %s (ID: %d): telegram=%v, webhook=%v", user.ID, sub.Name, sub.ID, telegramErr, webhookErr)
				continue
			}

			now := time.Now()
			sub.LastCancellationReminderSent = &now
			if sub.CancellationDate != nil {
				cancellationDateCopy := *sub.CancellationDate
				sub.LastCancellationReminderDate = &cancellationDateCopy
			}
			if _, updateErr := userSubscriptions.Update(sub.ID, sub); updateErr != nil {
				log.Printf("Warning: Failed to update last cancellation reminder sent for user %d subscription %s (ID: %d): %v", user.ID, sub.Name, sub.ID, updateErr)
			}
		}
	}
}
