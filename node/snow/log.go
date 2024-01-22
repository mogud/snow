package snow

func (ss *Service) Tracef(format string, args ...any) {
	ss.logger.Tracef(format, args...)
}

func (ss *Service) Debugf(format string, args ...any) {
	ss.logger.Debugf(format, args...)
}

func (ss *Service) Infof(format string, args ...any) {
	ss.logger.Infof(format, args...)
}

func (ss *Service) Warnf(format string, args ...any) {
	ss.logger.Warnf(format, args...)
}

func (ss *Service) Errorf(format string, args ...any) {
	ss.logger.Errorf(format, args...)
}

func (ss *Service) Fatalf(format string, args ...any) {
	ss.logger.Fatalf(format, args...)
}
