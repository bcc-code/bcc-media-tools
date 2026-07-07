package apiv1

import "slices"

func (p *Permissions) CanUpload() bool {
	// If the users is admin, bmmm admin, or has languages assigned
	return p.Admin ||
		(p.Bmm != nil && (p.Bmm.Admin || (p.Bmm.Languages != nil && len(p.Bmm.Languages) > 0)))
}

// CanExport reports whether the user may reach the export tool at all (global
// admin, export admin, at least one allowed destination, or the dedicated
// timed-metadata permission). Note: this only gates access to the tool/config;
// starting a destination export is still authorized per-destination via
// CanExportTo, and timed metadata via CanExportTimedMetadata.
func (p *Permissions) CanExport() bool {
	return p.Admin ||
		(p.Export != nil && (p.Export.Admin || len(p.Export.Destinations) > 0 || p.Export.TimedMetadata))
}

// CanExportTo reports whether the user may export to a specific destination.
func (p *Permissions) CanExportTo(destination string) bool {
	return p.Admin ||
		(p.Export != nil && (p.Export.Admin || slices.Contains(p.Export.Destinations, destination)))
}

// CanExportTimedMetadata reports whether the user may trigger the (VOD-affecting)
// timed metadata export. This is a dedicated permission, not implied by general
// export access.
func (p *Permissions) CanExportTimedMetadata() bool {
	return p.Admin ||
		(p.Export != nil && (p.Export.Admin || p.Export.TimedMetadata))
}

// CanBulkExport reports whether the user may use the bulk export flow (paste a
// list of VX-ids and export them all). This is a dedicated permission, not
// implied by general export access.
func (p *Permissions) CanBulkExport() bool {
	return p.Admin ||
		(p.Export != nil && p.Export.BulkExport)
}

// CanVBExport reports whether the user may use the VB export tool at all.
func (p *Permissions) CanVBExport() bool {
	return p.Admin ||
		(p.VbExport != nil && (p.VbExport.Admin || len(p.VbExport.Destinations) > 0))
}

// CanBulkVBExport reports whether the user may use the bulk VB export flow.
func (p *Permissions) CanBulkVBExport() bool {
	return p.Admin ||
		(p.VbExport != nil && p.VbExport.BulkExport)
}

// CanVBExportTo reports whether the user may VB-export to a specific destination.
func (p *Permissions) CanVBExportTo(destination string) bool {
	return p.Admin ||
		(p.VbExport != nil && (p.VbExport.Admin || slices.Contains(p.VbExport.Destinations, destination)))
}

// CanCantemoPreview reports whether the user may trigger preview generation.
func (p *Permissions) CanCantemoPreview() bool {
	return p.Admin || (p.Cantemo != nil && p.Cantemo.Preview)
}

// CanCantemoTranscribe reports whether the user may trigger transcription.
func (p *Permissions) CanCantemoTranscribe() bool {
	return p.Admin || (p.Cantemo != nil && p.Cantemo.Transcribe)
}

// CanCantemoSubtitles reports whether the user may update subtitles from Subtrans.
func (p *Permissions) CanCantemoSubtitles() bool {
	return p.Admin || (p.Cantemo != nil && p.Cantemo.Subtitles)
}

// CanCantemoRelations reports whether the user may update asset relations.
func (p *Permissions) CanCantemoRelations() bool {
	return p.Admin || (p.Cantemo != nil && p.Cantemo.Relations)
}

// CanVault reports whether the user may use the VAULT search view.
func (p *Permissions) CanVault() bool {
	return p.Admin || (p.Vault != nil && p.Vault.Enabled)
}

// CanViewJobs reports whether the user may view the jobs dashboard: admins, or
// anyone with access to a tool that produces workflows (export, VB export, BMM
// upload, transcription) or the vault. Gated the same tool-by-tool way as the
// rest of the app rather than being open to any authenticated user.
func (p *Permissions) CanViewJobs() bool {
	return p.Admin ||
		p.CanExport() ||
		p.CanVBExport() ||
		p.CanUpload() ||
		(p.Bmm != nil && len(p.Bmm.Podcasts) > 0) ||
		(p.Transcription != nil && (p.Transcription.Admin || p.Transcription.Mediabanken)) ||
		p.CanVault()
}
