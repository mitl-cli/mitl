// Package templates provides Dockerfile templates for various stacks.
// Keep templates separate for maintainability.
package templates

// PHPTemplates for different PHP frameworks
var PHPTemplates = map[string]string{
	"generic": PHPGenericTemplate,
}

// NodeTemplates for different Node frameworks
var NodeTemplates = map[string]string{
	"generic": NodeGenericTemplate,
}

// PHPGenericTemplate for generic PHP applications
const PHPGenericTemplate = `# PHP Generic Application
FROM php:{{.Version}}-fpm-alpine

RUN docker-php-ext-install {{.Extensions}}

WORKDIR /app
COPY . .

{{if .UseComposer}}
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer install --no-dev --optimize-autoloader
{{end}}

CMD ["php-fpm"]
`

// NodeGenericTemplate for generic Node app (simple)
const NodeGenericTemplate = `# Node Generic Application
FROM node:{{.NodeVersion}}-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
EXPOSE 3000
CMD ["npm", "start"]
`
