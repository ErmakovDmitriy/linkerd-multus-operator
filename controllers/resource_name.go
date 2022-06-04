package controllers

const NamePrefix string = "linkerd-"

// computeMultusNetAttachResourceName used to generate a name for Multus
// NetAttachDefinition.
func computeMultusNetAttachResourceName(linkerdAttachName string) string {
	return NamePrefix + linkerdAttachName
}
