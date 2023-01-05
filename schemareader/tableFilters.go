package schemareader

import (
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/uyuni-project/inter-server-sync/sqlUtil"
)

const (
	VirtualIndexName = "virtual_main_unique_index"
)

func applyTableFilters(table Table) Table {
	switch table.Name {
	case "rhnchecksumtype":
		table.PKSequence = "rhn_checksum_id_seq"
	case "rhnchecksum":
		table.PKSequence = "rhnchecksum_seq"
	case "rhnpackagearch":
		table.PKSequence = "rhn_package_arch_id_seq"
	case "rhnchannelarch":
		table.PKSequence = "rhn_channel_arch_id_seq"
	case "rhnpackagename":
		// constraint: rhn_pn_id_pk
		table.PKSequence = "RHN_PKG_NAME_SEQ"
	case "rhnpackagenevra":
		table.PKSequence = "rhn_pkgnevra_id_seq"
	case "rhnpackagesource":
		table.PKSequence = "rhn_package_source_id_seq"
	case "rhnpackagekey":
		table.PKSequence = "rhn_pkey_id_seq"
	case "rhnpackageextratag":
		virtualIndexColumns := []string{"package_id", "key_id"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "rhnpackageevr":
		// constraint: rhn_pe_id_pk
		table.PKSequence = "rhn_pkg_evr_seq"
		unexportColumns := make(map[string]bool)
		unexportColumns["type"] = true
		table.UnexportColumns = unexportColumns
		table.UniqueIndexes["rhn_pe_v_r_e_uq"] = UniqueIndex{Name: "rhn_pe_v_r_e_uq",
			Columns: append(table.UniqueIndexes["rhn_pe_v_r_e_uq"].Columns, "type")}
		table.UniqueIndexes["rhn_pe_v_r_uq"] = UniqueIndex{Name: "rhn_pe_v_r_uq",
			Columns: append(table.UniqueIndexes["rhn_pe_v_r_uq"].Columns, "type")}
	case "rhnpackage":
		// We need to add a virtual unique constraint
		table.PKSequence = "RHN_PACKAGE_ID_SEQ"
		virtualIndexColumns := []string{"name_id", "evr_id", "package_arch_id", "checksum_id", "org_id"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "rhnpackagechangelogdata":
		// We need to add a virtual unique constraint
		table.PKSequence = "rhn_pkg_cld_id_seq"
		virtualIndexColumns := []string{"name", "text", "time"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "rhnpackagechangelogrec":
		table.PKSequence = "rhn_pkg_cl_id_seq"
	case "rhnpackagecapability":
		// pkid: rhn_pkg_capability_id_pk
		table.PKSequence = "RHN_PKG_CAPABILITY_ID_SEQ"
		// table has real unique index, but they are complex and useless, since we do nothing in the conflict
		// to simplify the code we can create a virtual index that will insure all data exists as supposed
		virtualIndexColumns := []string{"name", "version"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "rhnconfigfiletype":
		virtualIndexColumns := []string{"label"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "rhnconfigfile":
		unexportColumns := make(map[string]bool)
		unexportColumns["latest_config_revision_id"] = true
		table.UnexportColumns = unexportColumns
	case "rhnconfigcontent":
		virtualIndexColumns := []string{"contents", "file_size", "checksum_id", "is_binary", "delim_start", "delim_end", "created"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "suseimageinfo":
		unexportColumns := make(map[string]bool)
		// Ignore actions relevant only to source server
		unexportColumns["build_action_id"] = true
		unexportColumns["inspect_action_id"] = true
		unexportColumns["build_server_id"] = true
		unexportColumns["log"] = true
		table.UnexportColumns = unexportColumns
		// Unfortunately images have only ID unique and that is not enough for our guessing game.
		// Create virtual compound index then as close as we can get
		virtualIndexColumns := []string{"name", "version", "image_type", "image_arch_id", "org_id", "curr_revision_num"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "suseimageinfochannel":
		virtualIndexColumns := []string{"channel_id", "image_info_id"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "suseimageprofile":
		table.PKSequence = "suse_imgprof_prid_seq"
		// rhnregtoken is completely non-unique standalone, use rhnactivation key instead as reference to the same id
		references := make([]Reference, 0)
		for _, r := range table.References {
			if strings.Compare(r.TableName, "rhnregtoken") == 0 {
				ref := Reference{}
				ref.TableName = "rhnactivationkey"
				ref.ColumnMapping = map[string]string{
					"token_id": "reg_token_id",
				}
				references = append(references, ref)
			} else {
				references = append(references, r)
			}
		}
		table.References = references
	case "susekiwiprofile":
		virtualIndexColumns := []string{"profile_id"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "susedockerfileprofile":
		virtualIndexColumns := []string{"profile_id", "path"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "rhnerrata":
		// this table has two unique indexes with the same size which can be used
		// we are fixing the usage to one of them to make it deterministic
		table.MainUniqueIndexName = "rhn_errata_adv_org_uq"
		table.RowModCallback = func(value []sqlUtil.RowDataStructure, table Table) []sqlUtil.RowDataStructure {
			for i, row := range value {
				if strings.Compare(row.ColumnName, "severity_id") == 0 {
					value[i].Value = value[i].GetInitialValue()
				}
			}
			return value
		}
	case "susesaltpillar":
		table.RowModCallback = func(value []sqlUtil.RowDataStructure, table Table) []sqlUtil.RowDataStructure {
			isImagePillar := false
			pillarColumn := 0
			for i, column := range value {
				if strings.Compare(column.ColumnName, "category") == 0 &&
					strings.HasPrefix(column.Value.(string), "Image") {
					log.Trace().Msgf("Updating pillar URLs of %s", column.Value)
					isImagePillar = true
				} else if strings.Compare(column.ColumnName, "pillar") == 0 {
					pillarColumn = i
				}
			}
			if isImagePillar {
				re := regexp.MustCompile(`https://[^/]+/os-images/`)
				repl := []byte("https://{SERVER_FQDN}/os-images/")
				value[pillarColumn].Value = re.ReplaceAll(value[pillarColumn].Value.([]byte), repl)
			}
			return value
		}
		virtualIndexColumns := []string{"server_id", "group_id", "org_id", "category"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	case "suseimagefile":
		table.PKSequence = "suse_image_file_id_seq"
		virtualIndexColumns := []string{"image_info_id", "file"}
		table.UniqueIndexes[VirtualIndexName] = UniqueIndex{Name: VirtualIndexName, Columns: virtualIndexColumns}
		table.MainUniqueIndexName = VirtualIndexName
	}
	return table
}
