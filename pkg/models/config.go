package models

type Config struct {
	APPName  string `default:"app name"`
	Contacts []struct {
		Name  string
		Email string `required:"true"`
	}
}
