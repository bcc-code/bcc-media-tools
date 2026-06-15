package apiv1

import "slices"

func (p *Permissions) CanUpload() bool {
	// If the users is admin, bmmm admin, or has languages assigned
	return p.Admin ||
		(p.Bmm != nil && (p.Bmm.Admin || (p.Bmm.Languages != nil && len(p.Bmm.Languages) > 0)))
}

// CanExport reports whether the user may use the export tool at all (global
// admin, export admin, or at least one allowed destination).
func (p *Permissions) CanExport() bool {
	return p.Admin ||
		(p.Export != nil && (p.Export.Admin || len(p.Export.Destinations) > 0))
}

// CanExportTo reports whether the user may export to a specific destination.
func (p *Permissions) CanExportTo(destination string) bool {
	return p.Admin ||
		(p.Export != nil && (p.Export.Admin || slices.Contains(p.Export.Destinations, destination)))
}
