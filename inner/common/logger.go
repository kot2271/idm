package common

import (
	"encoding/json"

	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger структура логгера
type Logger struct {
	*zap.Logger
}

// NewLogger функция-конструктор логгера
func NewLogger(cfg Config) *Logger {
	var zapEncoderCfg = zapcore.EncoderConfig{
		TimeKey:          "timestamp",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.LowercaseLevelEncoder,
		EncodeTime:       zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000000"),
		EncodeDuration:   zapcore.MillisDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: "  ",
	}
	var zapCfg = zap.Config{
		Level:       zap.NewAtomicLevelAt(parseLogLevel(cfg.LogLevel)),
		Development: cfg.LogDevelopMode,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		// пишем записи в формате JSON
		Encoding:      "json",
		EncoderConfig: zapEncoderCfg,
		// логируем сообщения и ошибки в консоль
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
	}
	var logger = zap.Must(zapCfg.Build())
	logger.Info("logger construction succeeded")
	var created = &Logger{logger}
	created.setNewFiberZapLogger()
	return created
}

// setNewFiberZapLogger устанавливает логгер для fiber
func (l *Logger) setNewFiberZapLogger() {
	var fiberzapLogger = fiberzap.NewLogger(fiberzap.LoggerConfig{
		SetLogger: l.Logger,
	})
	log.SetLogger(fiberzapLogger)
}

// ParseRequestBody парсит тело запроса и возвращает структурированные поля для логирования
// Экспортируем для использования в middleware
func ParseRequestBody(bodyData []byte) []zap.Field {
	var requestData map[string]any
	if err := json.Unmarshal(bodyData, &requestData); err != nil {
		// Если не удается распарсить JSON, логируем как есть
		return []zap.Field{zap.String("body", string(bodyData))}
	}

	var fields []zap.Field

	// Извлекаем поля для employee
	if name, ok := requestData["name"].(string); ok {
		fields = append(fields, zap.String("name", name))
	}
	if email, ok := requestData["email"].(string); ok {
		fields = append(fields, zap.String("email", email))
	}
	if position, ok := requestData["position"].(string); ok {
		fields = append(fields, zap.String("position", position))
	}
	if department, ok := requestData["department"].(string); ok {
		fields = append(fields, zap.String("department", department))
	}
	if roleId, ok := requestData["role_id"].(float64); ok {
		fields = append(fields, zap.Int64("role_id", int64(roleId)))
	}

	return fields
}

// parseLogLevel парсит уровень логирования из строки в zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug", "DEBUG":
		return zapcore.DebugLevel
	case "info", "INFO":
		return zapcore.InfoLevel
	case "warn", "WARN":
		return zapcore.WarnLevel
	case "error", "ERROR":
		return zapcore.ErrorLevel
	case "panic", "PANIC":
		return zapcore.PanicLevel
	case "fatal", "FATAL":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// getRequestID извлекает requestId из контекста fiber
func (l *Logger) getRequestID(ctx *fiber.Ctx) string {
	requestID := ctx.Get("X-Request-ID")
	if requestID == "" {
		if reqID := ctx.Locals("requestid"); reqID != nil {
			if id, ok := reqID.(string); ok {
				requestID = id
			}
		}
	}
	return requestID
}

// InfoCtx логирует сообщение уровня Info с requestId из контекста
func (l *Logger) InfoCtx(ctx *fiber.Ctx, msg string, fields ...zap.Field) {
	requestID := l.getRequestID(ctx)
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	l.Info(msg, fields...)
}

// DebugCtx логирует сообщение уровня Debug с requestId из контекста
func (l *Logger) DebugCtx(ctx *fiber.Ctx, msg string, fields ...zap.Field) {
	requestID := l.getRequestID(ctx)
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	l.Debug(msg, fields...)
}

// ErrorCtx логирует сообщение уровня Error с requestId из контекста
func (l *Logger) ErrorCtx(ctx *fiber.Ctx, msg string, fields ...zap.Field) {
	requestID := l.getRequestID(ctx)
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	l.Error(msg, fields...)
}

// WarnCtx логирует сообщение уровня Warn с requestId из контекста
func (l *Logger) WarnCtx(ctx *fiber.Ctx, msg string, fields ...zap.Field) {
	requestID := l.getRequestID(ctx)
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	l.Warn(msg, fields...)
}

// FatalCtx логирует сообщение уровня Fatal с requestId из контекста
func (l *Logger) FatalCtx(ctx *fiber.Ctx, msg string, fields ...zap.Field) {
	requestID := l.getRequestID(ctx)
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	l.Fatal(msg, fields...)
}

// PanicCtx логирует сообщение уровня Panic с requestId из контекста
func (l *Logger) PanicCtx(ctx *fiber.Ctx, msg string, fields ...zap.Field) {
	requestID := l.getRequestID(ctx)
	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	l.Panic(msg, fields...)
}
