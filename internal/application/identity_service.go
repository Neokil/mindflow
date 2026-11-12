package application

import (
	"context"

	"mindflow/internal/domain"
)

func (s *Service) Setup(ctx context.Context, actor domain.Principal, v domain.SetupCommand) (Completed, error) {
	return s.repo.Setup(ctx, actor, v)
}

func (s *Service) SetupTestEmail(ctx context.Context, actor domain.Principal, v domain.SetupTestEmailCommand) (Completed, error) {
	return s.repo.SetupTestEmail(ctx, actor, v)
}

func (s *Service) Login(ctx context.Context, actor domain.Principal, v domain.LoginCommand) (domain.LoginResult, error) {
	return s.repo.Login(ctx, actor, v)
}

func (s *Service) ListUsers(ctx context.Context, actor domain.Principal) ([]UserDelivery, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return nil, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.ListUsers(ctx, actor)
}

func (s *Service) Invite(ctx context.Context, actor domain.Principal, v domain.InviteCommand) (Created, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return Created{}, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.Invite(ctx, actor, v)
}

func (s *Service) ResendInvite(ctx context.Context, actor domain.Principal, id string) (Completed, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return Completed{}, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.ResendInvite(ctx, actor, id)
}

func (s *Service) AdminReset(ctx context.Context, actor domain.Principal, id string) (Completed, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return Completed{}, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.AdminReset(ctx, actor, id)
}

func (s *Service) Forgot(ctx context.Context, actor domain.Principal, v domain.ForgotCommand) (Completed, error) {
	return s.repo.Forgot(ctx, actor, v)
}

func (s *Service) Redeem(ctx context.Context, actor domain.Principal, purpose string, v domain.RedeemCommand) (Completed, error) {
	return s.repo.Redeem(ctx, actor, purpose, v)
}

func (s *Service) UpdateUser(ctx context.Context, actor domain.Principal, id string, v domain.UpdateUserCommand) (Completed, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return Completed{}, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.UpdateUser(ctx, actor, id, v)
}

func (s *Service) DeleteUser(ctx context.Context, actor domain.Principal, id string, v domain.DeleteUserCommand) (Completed, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return Completed{}, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.DeleteUser(ctx, actor, id, v)
}

func (s *Service) UpdateSettings(ctx context.Context, actor domain.Principal, v domain.UpdateSettingsCommand) (Completed, error) {
	if actor.Role != "admin" || actor.Status != "active" {
		return Completed{}, domain.Fault("forbidden", "Administrator access required.")
	}

	return s.repo.UpdateSettings(ctx, actor, v)
}

func (s *Service) Initialized(ctx context.Context) (bool, error) { return s.repo.Initialized(ctx) }

func (s *Service) Settings(ctx context.Context) (domain.Settings, error) { return s.repo.Settings(ctx) }

func (s *Service) Authenticate(ctx context.Context, cookie string) (domain.Principal, error) {
	return s.repo.Authenticate(ctx, cookie)
}

func (s *Service) Logout(ctx context.Context, actor domain.Principal) error {
	return s.repo.Logout(ctx, actor)
}

func (s *Service) SetAppearance(ctx context.Context, actor domain.Principal, value string) error {
	return s.repo.SetAppearance(ctx, actor, value)
}

func (s *Service) SendTestEmail(ctx context.Context, actor domain.Principal) error {
	if actor.Role != "admin" || actor.Status != "active" {
		return domain.Fault("forbidden", "Administrator access required.")
	}
	settings, err := s.repo.Settings(ctx)
	if err != nil {
		return err
	}
	if err = s.repo.SendMail(ctx, settings.SMTP, actor.Email, "mindflow test email", "Your mindflow email settings are working."); err != nil {
		return domain.Invalid("The SMTP test failed. Check the host, credentials, and TLS settings.")
	}
	return nil
}
