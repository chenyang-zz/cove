package tool

func cloneDescriptor(descriptor Descriptor) Descriptor {
	descriptor.InputSchema = cloneMap(descriptor.InputSchema)
	descriptor.OutputSchema = cloneMap(descriptor.OutputSchema)
	descriptor.Annotations = cloneMap(descriptor.Annotations)
	return descriptor
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
