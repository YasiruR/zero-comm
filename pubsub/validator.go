package pubsub

//type validator struct {
//	hashes *sync.Map
//	log    log.Logger
//}
//
//func newValidator(l log.Logger) *validator {
//	return &validator{
//		hashes: &sync.Map{},
//		log:    l,
//	}
//}
//
//// updateHash should be called whenever the group state is updated
//// with respect to status or role. A hasher is initiated as per each
//// update request to be on the safe side with concurrent writes.
//func (v *validator) updateHash(topic string, grp []models.Member) error {
//	val, err := v.calculate(grp)
//	if err != nil {
//		return fmt.Errorf(`calculating hash value failed - %v`, err)
//	}
//
//	v.hashes.Store(topic, val)
//	v.log.Trace(fmt.Sprintf(`group checksum updated to %s`, val))
//	return nil
//}
//
//func (v *validator) calculate(grp []models.Member) (string, error) {
//	byts, err := json.Marshal(v.order(grp))
//	if err != nil {
//		return ``, fmt.Errorf(`marshalling group members failed - %v`, err)
//	}
//
//	h := sha256.New()
//	h.Write(byts)
//	val := make([]byte, len(byts))
//	base64.StdEncoding.Encode(val, h.Sum(nil))
//
//	return string(bytes.Trim(val, "\x00")), nil
//}
//
//func (v *validator) hash(topic string) (string, error) {
//	val, ok := v.hashes.Load(topic)
//	if !ok {
//		return ``, fmt.Errorf(`hash value for the topic %s does not exist`, topic)
//	}
//
//	str, ok := val.(string)
//	if !ok {
//		return ``, fmt.Errorf(`hash value type is incompatible (%v) - should be string`, val)
//	}
//
//	return str, nil
//}
//
//// verify takes a map of group-state hash values indexed by label and returns
//// the list of members deviated from the majority. In cases where multiple
//// intruder sets exist, the set with least number of deviated members is returned.
//func (v *validator) verify(states map[string]string) (invalidMems []string, ok bool) {
//	// inverse map of states where key is hash and value is the list of members
//	mems := make(map[string][]string)
//	for membr, val := range states {
//		if mems[val] == nil {
//			mems[val] = []string{}
//		}
//		mems[val] = append(mems[val], membr)
//	}
//
//	// choose the set with the least number of deviated members
//	deviated := len(states)
//	for _, membrs := range mems {
//		if len(membrs) < deviated {
//			deviated = len(membrs)
//			invalidMems = membrs
//		}
//	}
//
//	if deviated == len(states) {
//		v.log.Info(`checksum verification completed successfully`)
//		return nil, true
//	}
//
//	return invalidMems, false
//}
//
//// order can be used to maintain a consistent order in the member list for
//// computation of group checksum values.
//// NOTE: test the functionality and performance (multiple iterations of a map)
//func (v *validator) order(grp []models.Member) (sorted []models.Member) {
//	var labels []string
//	for _, m := range grp {
//		labels = append(labels, m.Label)
//	}
//
//	sort.Strings(labels)
//	for _, l := range labels {
//		for _, m := range grp {
//			if m.Label == l {
//				sorted = append(sorted, m)
//				break
//			}
//		}
//	}
//
//	return sorted
//}
