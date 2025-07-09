# Makefile для управления Swagger документацией

# Установка swag утилиты
install-swag:
	go install github.com/swaggo/swag/cmd/swag@latest

# Генерация Swagger документации
swagger-gen:
	swag init -d cmd,inner --parseDependency --parseInternal
# Форматирование Swagger комментариев
swagger-fmt:
	swag fmt -d cmd,inner

# Полная пересборка документации
swagger-rebuild: swagger-fmt swagger-gen
	@echo "Swagger документация успешно сгенерирована в папке docs/"
 

# Запуск приложения
run:
	go run cmd/main.go

# Сборка приложения
build:
	go build cmd/main.go

test:
	go test -v idm/...

.PHONY: install-swag swagger-gen swagger-fmt swagger-rebuild run build