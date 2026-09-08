package email

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

type Config struct {
	From     string
	SMTPHost string
	SMTPPort int
	Username string
	Password string
}

func SendVerification(to, token string, cfg Config) error {
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Подтверждение регистрации в NeeKoobaMeemo!")

	// Ссылка для подтверждения
	link := fmt.Sprintf("http://localhost:8081/verify?token=%s", token)
	body := fmt.Sprintf(`
        <h2>Добро пожаловать в NeeKoobaMeemo!</h2>
        <p>Перейдите по ссылке, чтобы подтвердить email:</p>
        <a href="%s">%s</a>
        <p>Если вы не регистрировались, просто проигнорируйте это письмо.</p>
    `, link, link)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(cfg.SMTPHost, cfg.SMTPPort, cfg.Username, cfg.Password)
	return d.DialAndSend(m)
}
