package apiv1

func (p *Permissions) CanUpload() bool {
	// If the users is admin, bmmm admin, or has languages assigned
	return p.Admin ||
		(p.Bmm != nil && (p.Bmm.Admin || (p.Bmm.Languages != nil && len(p.Bmm.Languages) > 0)))
}
