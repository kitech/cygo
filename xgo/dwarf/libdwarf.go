package dwarf

/*
#cgo LDFLAGS: -ldwarf

#include <execinfo.h>
#include <libdwarf/dwarf.h>
#include <libdwarf/libdwarf.h>
#include <libelf.h>

*/
import "C"

type Unsigned uint64
type Signed int64
type Off uint64
type Addr uint64
type Bool int
type Half uint16
type Small uint8
type Ptr voidptr

type FormData16 struct {
	fd_data [16]byte
}

type Sig8 struct {
	signature [8]byte
}

type Block struct {
	bl_len  Unsigned
	bl_data Ptr

	bl_from_loclist   Small
	bl_section_offset Unsigned
}

type Loc struct {
	lr_atom    Small
	lr_number  Unsigned
	lr_number2 Unsigned
	lr_offset  Unsigned
}

type Locdesc struct {
	ld_lopc           Addr
	ld_hipc           Addr
	ld_cents          Half
	ld_s              *Loc
	ld_from_loclist   Small
	ld_section_offset Unsigned
}

func init() {
	// assert(sizeof(FormData16) == sizeof(C.Dwarf_Form_Data16))
	assert(sizeof(Sig8) == sizeof(C.Dwarf_Sig8))
	assert(sizeof(Block) == sizeof(C.Dwarf_Block))
	assert(sizeof(Loc) == sizeof(C.Dwarf_Loc))
	assert(sizeof(Locdesc) == sizeof(C.Dwarf_Locdesc))
}

const (
	DW_RANGES_ENTRY = iota
	DW_RANGES_ADDRESS_SELECTION
	DW_RANGES_END
)

type Ranges struct {
	dwr_addr1 Addr
	dwr_addr2 Addr
	dwr_type  int
}

type FrameOp struct {
	fp_base_op     Small
	fp_extended_op Small
	fp_register    Half

	/*  Value may be signed, depends on op.
	    Any applicable data_alignment_factor has
	    not been applied, this is the  raw offset. */
	fp_offset       Unsigned
	fp_instr_offset Off
}

const DW_REG_TABLE_SIZE = 66

const DW_FRAME_CFA_COL3 = 1436

type RegtableEntry struct {
	dw_offset_relevant Small

	/* For DWARF2, always 0 */
	dw_value_type Small
	dw_regnum     Half

	/*  The data type here should  the larger of Dwarf_Addr
	    and Dwarf_Unsigned and Dwarf_Signed. */
	dw_offset Addr
}

type Regtable struct {
	rules [DW_REG_TABLE_SIZE]RegtableEntry
}

type RegtableEntry3 struct {
	dw_offset_relevant     Small
	dw_value_type          Small
	dw_regnum              Half
	dw_offset_or_block_len Unsigned
	dw_block_ptr           Ptr
}

type Regtable3 struct {
	rt3_cfa_rule       RegtableEntry3
	rt3_reg_table_size Half
	rt3_rules          *RegtableEntry3
}

type PMarker struct {
	ma_marker Unsigned
	ma_offset Unsigned
}

type RelocationData struct {
	drd_type byte /* Cast to/from Dwarf_Rel_Type
	   to keep size small in struct. */
	drd_length byte /* Length in bytes of data being
	   relocated. 4 for 32bit data,
	   8 for 64bit data. */
	drd_offset       Unsigned /* Where the data to reloc is. */
	drd_symbol_index Unsigned
}

type PStringAttr struct {
	sa_offset Unsigned /* Offset of string attribute data */
	sa_nbytes Unsigned
}

type Debug voidptr
type Die voidptr
type Line voidptr
type Global voidptr
type Func voidptr
type Type voidptr
type Var voidptr
type Weak voidptr
type Error voidptr
type Attribute voidptr
type Abbrev voidptr
type Fde voidptr
type Cie voidptr
type Arange voidptr
type Gdbindex voidptr
type LineContext voidptr

type Tag uint64

type Handler func(dwerr Error, errarg Ptr)

type ObjAccessSection struct {
	/*  addr is the virtual address of the first byte of
	    the section data.  Usually zero when the address
	    makes no sense for a given section. */
	addr Addr

	/* Section type. */
	type_ Unsigned

	/* Size in bytes of the section. */
	size Unsigned

	/*  Having an accurate section name makes debugging of libdwarf easier.
	    and is essential to find the .debug_ sections.  */
	name byteptr
	/*  Set link to zero if it is meaningless.  If non-zero
	    it should be a link to a rela section or from symtab
	    to strtab.  In Elf it is sh_link. */
	link Unsigned

	/*  The section header index of the section to which the
	    relocation applies. In Elf it is sh_info. */
	info Unsigned

	/*  Elf sections that are tables have a non-zero entrysize so
	    the count of entries can be calculated even without
	    the right structure definition. If your object format
	    does not have this data leave this zero. */
	entrysize Unsigned
}

type ObjAccessMethods struct {
	// TODO compiler
	// get_section_info func(obj voidptr, section_index Half, return_section *DwarfObjAccessSection, dwerr *int) int
}

type ObjAccessInterface struct {
	/*  object is a void* as it hides the data the object access routines
	    need (which varies by library in use and object format).
	*/
	object  voidptr
	methods *ObjAccessMethods
}

/*
   Dwarf_dealloc() alloc_type arguments.
   Argument points to:
*/
const DW_DLA_STRING = 0x01      /* char* */
const DW_DLA_LOC = 0x02         /* Dwarf_Loc */
const DW_DLA_LOCDESC = 0x03     /* Dwarf_Locdesc */
const DW_DLA_ELLIST = 0x04      /* Dwarf_Ellist (not used)*/
const DW_DLA_BOUNDS = 0x05      /* Dwarf_Bounds (not used) */
const DW_DLA_BLOCK = 0x06       /* Dwarf_Block */
const DW_DLA_DEBUG = 0x07       /* Dwarf_Debug */
const DW_DLA_DIE = 0x08         /* Dwarf_Die */
const DW_DLA_LINE = 0x09        /* Dwarf_Line */
const DW_DLA_ATTR = 0x0a        /* Dwarf_Attribute */
const DW_DLA_TYPE = 0x0b        /* Dwarf_Type	(not used) */
const DW_DLA_SUBSCR = 0x0c      /* Dwarf_Subscr (not used) */
const DW_DLA_GLOBAL = 0x0d      /* Dwarf_Global */
const DW_DLA_ERROR = 0x0e       /* Dwarf_Error */
const DW_DLA_LIST = 0x0f        /* a list */
const DW_DLA_LINEBUF = 0x10     /* Dwarf_Line* (not used) */
const DW_DLA_ARANGE = 0x11      /* Dwarf_Arange */
const DW_DLA_ABBREV = 0x12      /* Dwarf_Abbrev */
const DW_DLA_FRAME_OP = 0x13    /* Dwarf_Frame_Op */
const DW_DLA_CIE = 0x14         /* Dwarf_Cie */
const DW_DLA_FDE = 0x15         /* Dwarf_Fde */
const DW_DLA_LOC_BLOCK = 0x16   /* Dwarf_Loc */
const DW_DLA_FRAME_BLOCK = 0x17 /* Dwarf_Frame Block (not used) */
const DW_DLA_FUNC = 0x18        /* Dwarf_Func */
const DW_DLA_TYPENAME = 0x19    /* Dwarf_Type */
const DW_DLA_VAR = 0x1a         /* Dwarf_Var */
const DW_DLA_WEAK = 0x1b        /* Dwarf_Weak */
const DW_DLA_ADDR = 0x1c        /* Dwarf_Addr sized entries */
const DW_DLA_RANGES = 0x1d      /* Dwarf_Ranges */

/* 0x1e (30) to 0x36 (54) reserved for internal to libdwarf types. */

const DW_DLA_GDBINDEX = 0x37      /* Dwarf_Gdbindex */
const DW_DLA_XU_INDEX = 0x38      /* Dwarf_Xu_Index_Header */
const DW_DLA_LOC_BLOCK_C = 0x39   /* Dwarf_Loc_c*/
const DW_DLA_LOCDESC_C = 0x3a     /* Dwarf_Locdesc_c */
const DW_DLA_LOC_HEAD_C = 0x3b    /* Dwarf_Loc_Head_c */
const DW_DLA_MACRO_CONTEXT = 0x3c /* Dwarf_Macro_Context */
/*  0x3d (61) is for libdwarf internal use.               */
const DW_DLA_DSC_HEAD = 0x3e    /* Dwarf_Dsc_Head */
const DW_DLA_DNAMES_HEAD = 0x3f /* Dwarf_Dnames_Head */
const DW_DLA_STR_OFFSETS = 0x40 /* struct Dwarf_Str_Offsets_Table_s */

/* The augmenter string for CIE */
const DW_CIE_AUGMENTER_STRING_V0 = "z"

/* dwarf_init() access arguments
 */
const DW_DLC_READ = 0  /* read only access */
const DW_DLC_WRITE = 1 /* write only access */
const DW_DLC_RDWR = 2  /* read/write access NOT SUPPORTED*/

/* 64-bit address-size target */
const DW_DLC_SIZE_64 = 0x40000000

/* 32-bit address-size target */
const DW_DLC_SIZE_32 = 0x20000000

/* 64-bit offset-size DWARF offsets (else 32bit) */
const DW_DLC_OFFSET_SIZE_64 = 0x10000000

/* 32-bit offset-size ELF object (ELFCLASS32) */
const DW_DLC_ELF_OFFSET_SIZE_32 = 0x00400000

/* 64-bit offset-size ELF object (ELFCLASS64)  */
const DW_DLC_ELF_OFFSET_SIZE_64 = 0x00020000

/* dwarf_producer_init* access flag modifiers
   Some new April 2014.
   If DW_DLC_STREAM_RELOCATIONS is set the
   DW_DLC_ISA_* flags are ignored. See the Dwarf_Rel_Type enum.
*/

/* Old style Elf binary relocation (.rel) records. The default. */
const DW_DLC_STREAM_RELOCATIONS = 0x02000000

/* use 32-bit  sec  offsets */
const DW_DLC_OFFSET32 = 0x00010000

/* The following 3 are new sensible names.
Old names above with same values. */
/* use 64-bit sec offsets in ELF */
const DW_DLC_OFFSET64 = 0x10000000

/* use 4 for address_size */
const DW_DLC_POINTER32 = 0x20000000

/* use 8 for address_size */
const DW_DLC_POINTER64 = 0x40000000

/* Special for IRIX only */
/* use Elf 64bit offset headers and non-std IRIX 64bitoffset headers */
const DW_DLC_IRIX_OFFSET64 = 0x00200000

/*  Usable with assembly output because it is up to the producer to
    deal with locations in whatever manner the calling producer
    code wishes.  For example, when the libdwarf caller wishes
    to produce relocations differently than the binary
    relocation bits that libdwarf Stream Relocations generate.
*/
const DW_DLC_SYMBOLIC_RELOCATIONS = 0x04000000

const DW_DLC_TARGET_BIGENDIAN = 0x08000000    /* Big    endian target */
const DW_DLC_TARGET_LITTLEENDIAN = 0x00100000 /* Little endian target */

/* dwarf_pcline function, slide arguments
 */
const DW_DLS_BACKWARD = -1 /* slide backward to find line */
const DW_DLS_NOSLIDE = 0   /* match exactly without sliding */
const DW_DLS_FORWARD = 1   /* slide forward to find line */

/* libdwarf error numbers
 */
const DW_DLE_NE = 0       /* no error */
const DW_DLE_VMM = 1      /* dwarf format/library version mismatch */
const DW_DLE_MAP = 2      /* memory map failure */
const DW_DLE_LEE = 3      /* libelf error */
const DW_DLE_NDS = 4      /* no debug section */
const DW_DLE_NLS = 5      /* no line section */
const DW_DLE_ID = 6       /* invalid descriptor for query */
const DW_DLE_IOF = 7      /* I/O failure */
const DW_DLE_MAF = 8      /* memory allocation failure */
const DW_DLE_IA = 9       /* invalid argument */
const DW_DLE_MDE = 10     /* mangled debugging entry */
const DW_DLE_MLE = 11     /* mangled line number entry */
const DW_DLE_FNO = 12     /* file not open */
const DW_DLE_FNR = 13     /* file not a regular file */
const DW_DLE_FWA = 14     /* file open with wrong access */
const DW_DLE_NOB = 15     /* not an object file */
const DW_DLE_MOF = 16     /* mangled object file header */
const DW_DLE_EOLL = 17    /* end of location list entries */
const DW_DLE_NOLL = 18    /* no location list section */
const DW_DLE_BADOFF = 19  /* Invalid offset */
const DW_DLE_EOS = 20     /* end of section  */
const DW_DLE_ATRUNC = 21  /* abbreviations section appears truncated*/
const DW_DLE_BADBITC = 22 /* Address size passed to dwarf bad*/
/* It is not an allowed size (64 or 32) */
/* Error codes defined by the current Libdwarf Implementation. */
const DW_DLE_DBG_ALLOC = 23
const DW_DLE_FSTAT_ERROR = 24
const DW_DLE_FSTAT_MODE_ERROR = 25
const DW_DLE_INIT_ACCESS_WRONG = 26
const DW_DLE_ELF_BEGIN_ERROR = 27
const DW_DLE_ELF_GETEHDR_ERROR = 28
const DW_DLE_ELF_GETSHDR_ERROR = 29
const DW_DLE_ELF_STRPTR_ERROR = 30
const DW_DLE_DEBUG_INFO_DUPLICATE = 31
const DW_DLE_DEBUG_INFO_NULL = 32
const DW_DLE_DEBUG_ABBREV_DUPLICATE = 33
const DW_DLE_DEBUG_ABBREV_NULL = 34
const DW_DLE_DEBUG_ARANGES_DUPLICATE = 35
const DW_DLE_DEBUG_ARANGES_NULL = 36
const DW_DLE_DEBUG_LINE_DUPLICATE = 37
const DW_DLE_DEBUG_LINE_NULL = 38
const DW_DLE_DEBUG_LOC_DUPLICATE = 39
const DW_DLE_DEBUG_LOC_NULL = 40
const DW_DLE_DEBUG_MACINFO_DUPLICATE = 41
const DW_DLE_DEBUG_MACINFO_NULL = 42
const DW_DLE_DEBUG_PUBNAMES_DUPLICATE = 43
const DW_DLE_DEBUG_PUBNAMES_NULL = 44
const DW_DLE_DEBUG_STR_DUPLICATE = 45
const DW_DLE_DEBUG_STR_NULL = 46
const DW_DLE_CU_LENGTH_ERROR = 47
const DW_DLE_VERSION_STAMP_ERROR = 48
const DW_DLE_ABBREV_OFFSET_ERROR = 49
const DW_DLE_ADDRESS_SIZE_ERROR = 50
const DW_DLE_DEBUG_INFO_PTR_NULL = 51
const DW_DLE_DIE_NULL = 52
const DW_DLE_STRING_OFFSET_BAD = 53
const DW_DLE_DEBUG_LINE_LENGTH_BAD = 54
const DW_DLE_LINE_PROLOG_LENGTH_BAD = 55
const DW_DLE_LINE_NUM_OPERANDS_BAD = 56
const DW_DLE_LINE_SET_ADDR_ERROR = 57 /* No longer used. */
const DW_DLE_LINE_EXT_OPCODE_BAD = 58
const DW_DLE_DWARF_LINE_NULL = 59
const DW_DLE_INCL_DIR_NUM_BAD = 60
const DW_DLE_LINE_FILE_NUM_BAD = 61
const DW_DLE_ALLOC_FAIL = 62
const DW_DLE_NO_CALLBACK_FUNC = 63
const DW_DLE_SECT_ALLOC = 64
const DW_DLE_FILE_ENTRY_ALLOC = 65
const DW_DLE_LINE_ALLOC = 66
const DW_DLE_FPGM_ALLOC = 67
const DW_DLE_INCDIR_ALLOC = 68
const DW_DLE_STRING_ALLOC = 69
const DW_DLE_CHUNK_ALLOC = 70
const DW_DLE_BYTEOFF_ERR = 71
const DW_DLE_CIE_ALLOC = 72
const DW_DLE_FDE_ALLOC = 73
const DW_DLE_REGNO_OVFL = 74
const DW_DLE_CIE_OFFS_ALLOC = 75
const DW_DLE_WRONG_ADDRESS = 76
const DW_DLE_EXTRA_NEIGHBORS = 77
const DW_DLE_WRONG_TAG = 78
const DW_DLE_DIE_ALLOC = 79
const DW_DLE_PARENT_EXISTS = 80
const DW_DLE_DBG_NULL = 81
const DW_DLE_DEBUGLINE_ERROR = 82
const DW_DLE_DEBUGFRAME_ERROR = 83
const DW_DLE_DEBUGINFO_ERROR = 84
const DW_DLE_ATTR_ALLOC = 85
const DW_DLE_ABBREV_ALLOC = 86
const DW_DLE_OFFSET_UFLW = 87
const DW_DLE_ELF_SECT_ERR = 88
const DW_DLE_DEBUG_FRAME_LENGTH_BAD = 89
const DW_DLE_FRAME_VERSION_BAD = 90
const DW_DLE_CIE_RET_ADDR_REG_ERROR = 91
const DW_DLE_FDE_NULL = 92
const DW_DLE_FDE_DBG_NULL = 93
const DW_DLE_CIE_NULL = 94
const DW_DLE_CIE_DBG_NULL = 95
const DW_DLE_FRAME_TABLE_COL_BAD = 96
const DW_DLE_PC_NOT_IN_FDE_RANGE = 97
const DW_DLE_CIE_INSTR_EXEC_ERROR = 98
const DW_DLE_FRAME_INSTR_EXEC_ERROR = 99
const DW_DLE_FDE_PTR_NULL = 100
const DW_DLE_RET_OP_LIST_NULL = 101
const DW_DLE_LINE_CONTEXT_NULL = 102
const DW_DLE_DBG_NO_CU_CONTEXT = 103
const DW_DLE_DIE_NO_CU_CONTEXT = 104
const DW_DLE_FIRST_DIE_NOT_CU = 105
const DW_DLE_NEXT_DIE_PTR_NULL = 106
const DW_DLE_DEBUG_FRAME_DUPLICATE = 107
const DW_DLE_DEBUG_FRAME_NULL = 108
const DW_DLE_ABBREV_DECODE_ERROR = 109
const DW_DLE_DWARF_ABBREV_NULL = 110
const DW_DLE_ATTR_NULL = 111
const DW_DLE_DIE_BAD = 112
const DW_DLE_DIE_ABBREV_BAD = 113
const DW_DLE_ATTR_FORM_BAD = 114
const DW_DLE_ATTR_NO_CU_CONTEXT = 115
const DW_DLE_ATTR_FORM_SIZE_BAD = 116
const DW_DLE_ATTR_DBG_NULL = 117
const DW_DLE_BAD_REF_FORM = 118
const DW_DLE_ATTR_FORM_OFFSET_BAD = 119
const DW_DLE_LINE_OFFSET_BAD = 120
const DW_DLE_DEBUG_STR_OFFSET_BAD = 121
const DW_DLE_STRING_PTR_NULL = 122
const DW_DLE_PUBNAMES_VERSION_ERROR = 123
const DW_DLE_PUBNAMES_LENGTH_BAD = 124
const DW_DLE_GLOBAL_NULL = 125
const DW_DLE_GLOBAL_CONTEXT_NULL = 126
const DW_DLE_DIR_INDEX_BAD = 127
const DW_DLE_LOC_EXPR_BAD = 128
const DW_DLE_DIE_LOC_EXPR_BAD = 129
const DW_DLE_ADDR_ALLOC = 130
const DW_DLE_OFFSET_BAD = 131
const DW_DLE_MAKE_CU_CONTEXT_FAIL = 132
const DW_DLE_REL_ALLOC = 133
const DW_DLE_ARANGE_OFFSET_BAD = 134
const DW_DLE_SEGMENT_SIZE_BAD = 135
const DW_DLE_ARANGE_LENGTH_BAD = 136
const DW_DLE_ARANGE_DECODE_ERROR = 137
const DW_DLE_ARANGES_NULL = 138
const DW_DLE_ARANGE_NULL = 139
const DW_DLE_NO_FILE_NAME = 140
const DW_DLE_NO_COMP_DIR = 141
const DW_DLE_CU_ADDRESS_SIZE_BAD = 142
const DW_DLE_INPUT_ATTR_BAD = 143
const DW_DLE_EXPR_NULL = 144
const DW_DLE_BAD_EXPR_OPCODE = 145
const DW_DLE_EXPR_LENGTH_BAD = 146
const DW_DLE_MULTIPLE_RELOC_IN_EXPR = 147
const DW_DLE_ELF_GETIDENT_ERROR = 148
const DW_DLE_NO_AT_MIPS_FDE = 149
const DW_DLE_NO_CIE_FOR_FDE = 150
const DW_DLE_DIE_ABBREV_LIST_NULL = 151
const DW_DLE_DEBUG_FUNCNAMES_DUPLICATE = 152
const DW_DLE_DEBUG_FUNCNAMES_NULL = 153
const DW_DLE_DEBUG_FUNCNAMES_VERSION_ERROR = 154
const DW_DLE_DEBUG_FUNCNAMES_LENGTH_BAD = 155
const DW_DLE_FUNC_NULL = 156
const DW_DLE_FUNC_CONTEXT_NULL = 157
const DW_DLE_DEBUG_TYPENAMES_DUPLICATE = 158
const DW_DLE_DEBUG_TYPENAMES_NULL = 159
const DW_DLE_DEBUG_TYPENAMES_VERSION_ERROR = 160
const DW_DLE_DEBUG_TYPENAMES_LENGTH_BAD = 161
const DW_DLE_TYPE_NULL = 162
const DW_DLE_TYPE_CONTEXT_NULL = 163
const DW_DLE_DEBUG_VARNAMES_DUPLICATE = 164
const DW_DLE_DEBUG_VARNAMES_NULL = 165
const DW_DLE_DEBUG_VARNAMES_VERSION_ERROR = 166
const DW_DLE_DEBUG_VARNAMES_LENGTH_BAD = 167
const DW_DLE_VAR_NULL = 168
const DW_DLE_VAR_CONTEXT_NULL = 169
const DW_DLE_DEBUG_WEAKNAMES_DUPLICATE = 170
const DW_DLE_DEBUG_WEAKNAMES_NULL = 171
const DW_DLE_DEBUG_WEAKNAMES_VERSION_ERROR = 172
const DW_DLE_DEBUG_WEAKNAMES_LENGTH_BAD = 173
const DW_DLE_WEAK_NULL = 174
const DW_DLE_WEAK_CONTEXT_NULL = 175
const DW_DLE_LOCDESC_COUNT_WRONG = 176
const DW_DLE_MACINFO_STRING_NULL = 177
const DW_DLE_MACINFO_STRING_EMPTY = 178
const DW_DLE_MACINFO_INTERNAL_ERROR_SPACE = 179
const DW_DLE_MACINFO_MALLOC_FAIL = 180
const DW_DLE_DEBUGMACINFO_ERROR = 181
const DW_DLE_DEBUG_MACRO_LENGTH_BAD = 182
const DW_DLE_DEBUG_MACRO_MAX_BAD = 183
const DW_DLE_DEBUG_MACRO_INTERNAL_ERR = 184
const DW_DLE_DEBUG_MACRO_MALLOC_SPACE = 185
const DW_DLE_DEBUG_MACRO_INCONSISTENT = 186
const DW_DLE_DF_NO_CIE_AUGMENTATION = 187
const DW_DLE_DF_REG_NUM_TOO_HIGH = 188
const DW_DLE_DF_MAKE_INSTR_NO_INIT = 189
const DW_DLE_DF_NEW_LOC_LESS_OLD_LOC = 190
const DW_DLE_DF_POP_EMPTY_STACK = 191
const DW_DLE_DF_ALLOC_FAIL = 192
const DW_DLE_DF_FRAME_DECODING_ERROR = 193
const DW_DLE_DEBUG_LOC_SECTION_SHORT = 194
const DW_DLE_FRAME_AUGMENTATION_UNKNOWN = 195
const DW_DLE_PUBTYPE_CONTEXT = 196 /* Unused. */
const DW_DLE_DEBUG_PUBTYPES_LENGTH_BAD = 197
const DW_DLE_DEBUG_PUBTYPES_VERSION_ERROR = 198
const DW_DLE_DEBUG_PUBTYPES_DUPLICATE = 199
const DW_DLE_FRAME_CIE_DECODE_ERROR = 200
const DW_DLE_FRAME_REGISTER_UNREPRESENTABLE = 201
const DW_DLE_FRAME_REGISTER_COUNT_MISMATCH = 202
const DW_DLE_LINK_LOOP = 203
const DW_DLE_STRP_OFFSET_BAD = 204
const DW_DLE_DEBUG_RANGES_DUPLICATE = 205
const DW_DLE_DEBUG_RANGES_OFFSET_BAD = 206
const DW_DLE_DEBUG_RANGES_MISSING_END = 207
const DW_DLE_DEBUG_RANGES_OUT_OF_MEM = 208
const DW_DLE_DEBUG_SYMTAB_ERR = 209
const DW_DLE_DEBUG_STRTAB_ERR = 210
const DW_DLE_RELOC_MISMATCH_INDEX = 211
const DW_DLE_RELOC_MISMATCH_RELOC_INDEX = 212
const DW_DLE_RELOC_MISMATCH_STRTAB_INDEX = 213
const DW_DLE_RELOC_SECTION_MISMATCH = 214
const DW_DLE_RELOC_SECTION_MISSING_INDEX = 215
const DW_DLE_RELOC_SECTION_LENGTH_ODD = 216
const DW_DLE_RELOC_SECTION_PTR_NULL = 217
const DW_DLE_RELOC_SECTION_MALLOC_FAIL = 218
const DW_DLE_NO_ELF64_SUPPORT = 219
const DW_DLE_MISSING_ELF64_SUPPORT = 220
const DW_DLE_ORPHAN_FDE = 221
const DW_DLE_DUPLICATE_INST_BLOCK = 222
const DW_DLE_BAD_REF_SIG8_FORM = 223
const DW_DLE_ATTR_EXPRLOC_FORM_BAD = 224
const DW_DLE_FORM_SEC_OFFSET_LENGTH_BAD = 225
const DW_DLE_NOT_REF_FORM = 226
const DW_DLE_DEBUG_FRAME_LENGTH_NOT_MULTIPLE = 227
const DW_DLE_REF_SIG8_NOT_HANDLED = 228
const DW_DLE_DEBUG_FRAME_POSSIBLE_ADDRESS_BOTCH = 229
const DW_DLE_LOC_BAD_TERMINATION = 230
const DW_DLE_SYMTAB_SECTION_LENGTH_ODD = 231
const DW_DLE_RELOC_SECTION_SYMBOL_INDEX_BAD = 232
const DW_DLE_RELOC_SECTION_RELOC_TARGET_SIZE_UNKNOWN = 233
const DW_DLE_SYMTAB_SECTION_ENTRYSIZE_ZERO = 234
const DW_DLE_LINE_NUMBER_HEADER_ERROR = 235
const DW_DLE_DEBUG_TYPES_NULL = 236
const DW_DLE_DEBUG_TYPES_DUPLICATE = 237
const DW_DLE_DEBUG_TYPES_ONLY_DWARF4 = 238
const DW_DLE_DEBUG_TYPEOFFSET_BAD = 239
const DW_DLE_GNU_OPCODE_ERROR = 240
const DW_DLE_DEBUGPUBTYPES_ERROR = 241
const DW_DLE_AT_FIXUP_NULL = 242
const DW_DLE_AT_FIXUP_DUP = 243
const DW_DLE_BAD_ABINAME = 244
const DW_DLE_TOO_MANY_DEBUG = 245
const DW_DLE_DEBUG_STR_OFFSETS_DUPLICATE = 246
const DW_DLE_SECTION_DUPLICATION = 247
const DW_DLE_SECTION_ERROR = 248
const DW_DLE_DEBUG_ADDR_DUPLICATE = 249
const DW_DLE_DEBUG_CU_UNAVAILABLE_FOR_FORM = 250
const DW_DLE_DEBUG_FORM_HANDLING_INCOMPLETE = 251
const DW_DLE_NEXT_DIE_PAST_END = 252
const DW_DLE_NEXT_DIE_WRONG_FORM = 253
const DW_DLE_NEXT_DIE_NO_ABBREV_LIST = 254
const DW_DLE_NESTED_FORM_INDIRECT_ERROR = 255
const DW_DLE_CU_DIE_NO_ABBREV_LIST = 256
const DW_DLE_MISSING_NEEDED_DEBUG_ADDR_SECTION = 257
const DW_DLE_ATTR_FORM_NOT_ADDR_INDEX = 258
const DW_DLE_ATTR_FORM_NOT_STR_INDEX = 259
const DW_DLE_DUPLICATE_GDB_INDEX = 260
const DW_DLE_ERRONEOUS_GDB_INDEX_SECTION = 261
const DW_DLE_GDB_INDEX_COUNT_ERROR = 262
const DW_DLE_GDB_INDEX_COUNT_ADDR_ERROR = 263
const DW_DLE_GDB_INDEX_INDEX_ERROR = 264
const DW_DLE_GDB_INDEX_CUVEC_ERROR = 265
const DW_DLE_DUPLICATE_CU_INDEX = 266
const DW_DLE_DUPLICATE_TU_INDEX = 267
const DW_DLE_XU_TYPE_ARG_ERROR = 268
const DW_DLE_XU_IMPOSSIBLE_ERROR = 269
const DW_DLE_XU_NAME_COL_ERROR = 270
const DW_DLE_XU_HASH_ROW_ERROR = 271
const DW_DLE_XU_HASH_INDEX_ERROR = 272

/* ..._FAILSAFE_ERRVAL is an aid when out of memory. */
const DW_DLE_FAILSAFE_ERRVAL = 273
const DW_DLE_ARANGE_ERROR = 274
const DW_DLE_PUBNAMES_ERROR = 275
const DW_DLE_FUNCNAMES_ERROR = 276
const DW_DLE_TYPENAMES_ERROR = 277
const DW_DLE_VARNAMES_ERROR = 278
const DW_DLE_WEAKNAMES_ERROR = 279
const DW_DLE_RELOCS_ERROR = 280
const DW_DLE_ATTR_OUTSIDE_SECTION = 281
const DW_DLE_FISSION_INDEX_WRONG = 282
const DW_DLE_FISSION_VERSION_ERROR = 283
const DW_DLE_NEXT_DIE_LOW_ERROR = 284
const DW_DLE_CU_UT_TYPE_ERROR = 285
const DW_DLE_NO_SUCH_SIGNATURE_FOUND = 286
const DW_DLE_SIGNATURE_SECTION_NUMBER_WRONG = 287
const DW_DLE_ATTR_FORM_NOT_DATA8 = 288
const DW_DLE_SIG_TYPE_WRONG_STRING = 289
const DW_DLE_MISSING_REQUIRED_TU_OFFSET_HASH = 290
const DW_DLE_MISSING_REQUIRED_CU_OFFSET_HASH = 291
const DW_DLE_DWP_MISSING_DWO_ID = 292
const DW_DLE_DWP_SIBLING_ERROR = 293
const DW_DLE_DEBUG_FISSION_INCOMPLETE = 294
const DW_DLE_FISSION_SECNUM_ERR = 295
const DW_DLE_DEBUG_MACRO_DUPLICATE = 296
const DW_DLE_DEBUG_NAMES_DUPLICATE = 297
const DW_DLE_DEBUG_LINE_STR_DUPLICATE = 298
const DW_DLE_DEBUG_SUP_DUPLICATE = 299
const DW_DLE_NO_SIGNATURE_TO_LOOKUP = 300
const DW_DLE_NO_TIED_ADDR_AVAILABLE = 301
const DW_DLE_NO_TIED_SIG_AVAILABLE = 302
const DW_DLE_STRING_NOT_TERMINATED = 303
const DW_DLE_BAD_LINE_TABLE_OPERATION = 304
const DW_DLE_LINE_CONTEXT_BOTCH = 305
const DW_DLE_LINE_CONTEXT_INDEX_WRONG = 306
const DW_DLE_NO_TIED_STRING_AVAILABLE = 307
const DW_DLE_NO_TIED_FILE_AVAILABLE = 308
const DW_DLE_CU_TYPE_MISSING = 309
const DW_DLE_LLE_CODE_UNKNOWN = 310
const DW_DLE_LOCLIST_INTERFACE_ERROR = 311
const DW_DLE_LOCLIST_INDEX_ERROR = 312
const DW_DLE_INTERFACE_NOT_SUPPORTED = 313
const DW_DLE_ZDEBUG_REQUIRES_ZLIB = 314
const DW_DLE_ZDEBUG_INPUT_FORMAT_ODD = 315
const DW_DLE_ZLIB_BUF_ERROR = 316
const DW_DLE_ZLIB_DATA_ERROR = 317
const DW_DLE_MACRO_OFFSET_BAD = 318
const DW_DLE_MACRO_OPCODE_BAD = 319
const DW_DLE_MACRO_OPCODE_FORM_BAD = 320
const DW_DLE_UNKNOWN_FORM = 321
const DW_DLE_BAD_MACRO_HEADER_POINTER = 322
const DW_DLE_BAD_MACRO_INDEX = 323
const DW_DLE_MACRO_OP_UNHANDLED = 324
const DW_DLE_MACRO_PAST_END = 325
const DW_DLE_LINE_STRP_OFFSET_BAD = 326
const DW_DLE_STRING_FORM_IMPROPER = 327
const DW_DLE_ELF_FLAGS_NOT_AVAILABLE = 328
const DW_DLE_LEB_IMPROPER = 329
const DW_DLE_DEBUG_LINE_RANGE_ZERO = 330
const DW_DLE_READ_LITTLEENDIAN_ERROR = 331
const DW_DLE_READ_BIGENDIAN_ERROR = 332
const DW_DLE_RELOC_INVALID = 333
const DW_DLE_INFO_HEADER_ERROR = 334
const DW_DLE_ARANGES_HEADER_ERROR = 335
const DW_DLE_LINE_OFFSET_WRONG_FORM = 336
const DW_DLE_FORM_BLOCK_LENGTH_ERROR = 337
const DW_DLE_ZLIB_SECTION_SHORT = 338
const DW_DLE_CIE_INSTR_PTR_ERROR = 339
const DW_DLE_FDE_INSTR_PTR_ERROR = 340
const DW_DLE_FISSION_ADDITION_ERROR = 341
const DW_DLE_HEADER_LEN_BIGGER_THAN_SECSIZE = 342
const DW_DLE_LOCEXPR_OFF_SECTION_END = 343
const DW_DLE_POINTER_SECTION_UNKNOWN = 344
const DW_DLE_ERRONEOUS_XU_INDEX_SECTION = 345
const DW_DLE_DIRECTORY_FORMAT_COUNT_VS_DIRECTORIES_MISMATCH = 346
const DW_DLE_COMPRESSED_EMPTY_SECTION = 347
const DW_DLE_SIZE_WRAPAROUND = 348
const DW_DLE_ILLOGICAL_TSEARCH = 349
const DW_DLE_BAD_STRING_FORM = 350
const DW_DLE_DEBUGSTR_ERROR = 351
const DW_DLE_DEBUGSTR_UNEXPECTED_REL = 352
const DW_DLE_DISCR_ARRAY_ERROR = 353
const DW_DLE_LEB_OUT_ERROR = 354
const DW_DLE_SIBLING_LIST_IMPROPER = 355
const DW_DLE_LOCLIST_OFFSET_BAD = 356
const DW_DLE_LINE_TABLE_BAD = 357
const DW_DLE_DEBUG_LOClISTS_DUPLICATE = 358
const DW_DLE_DEBUG_RNGLISTS_DUPLICATE = 359
const DW_DLE_ABBREV_OFF_END = 360
const DW_DLE_FORM_STRING_BAD_STRING = 361
const DW_DLE_AUGMENTATION_STRING_OFF_END = 362
const DW_DLE_STRING_OFF_END_PUBNAMES_LIKE = 363
const DW_DLE_LINE_STRING_BAD = 364
const DW_DLE_DEFINE_FILE_STRING_BAD = 365
const DW_DLE_MACRO_STRING_BAD = 366
const DW_DLE_MACINFO_STRING_BAD = 367
const DW_DLE_ZLIB_UNCOMPRESS_ERROR = 368
const DW_DLE_IMPROPER_DWO_ID = 369
const DW_DLE_GROUPNUMBER_ERROR = 370
const DW_DLE_ADDRESS_SIZE_ZERO = 371
const DW_DLE_DEBUG_NAMES_HEADER_ERROR = 372
const DW_DLE_DEBUG_NAMES_AUG_STRING_ERROR = 373
const DW_DLE_DEBUG_NAMES_PAD_NON_ZERO = 374
const DW_DLE_DEBUG_NAMES_OFF_END = 375
const DW_DLE_DEBUG_NAMES_ABBREV_OVERFLOW = 376
const DW_DLE_DEBUG_NAMES_ABBREV_CORRUPTION = 377
const DW_DLE_DEBUG_NAMES_NULL_POINTER = 378
const DW_DLE_DEBUG_NAMES_BAD_INDEX_ARG = 379
const DW_DLE_DEBUG_NAMES_ENTRYPOOL_OFFSET = 380
const DW_DLE_DEBUG_NAMES_UNHANDLED_FORM = 381
const DW_DLE_LNCT_CODE_UNKNOWN = 382
const DW_DLE_LNCT_FORM_CODE_NOT_HANDLED = 383
const DW_DLE_LINE_HEADER_LENGTH_BOTCH = 384
const DW_DLE_STRING_HASHTAB_IDENTITY_ERROR = 385
const DW_DLE_UNIT_TYPE_NOT_HANDLED = 386
const DW_DLE_GROUP_MAP_ALLOC = 387
const DW_DLE_GROUP_MAP_DUPLICATE = 388
const DW_DLE_GROUP_COUNT_ERROR = 389
const DW_DLE_GROUP_INTERNAL_ERROR = 390
const DW_DLE_GROUP_LOAD_ERROR = 391
const DW_DLE_GROUP_LOAD_READ_ERROR = 392
const DW_DLE_AUG_DATA_LENGTH_BAD = 393
const DW_DLE_ABBREV_MISSING = 394
const DW_DLE_NO_TAG_FOR_DIE = 395
const DW_DLE_LOWPC_WRONG_CLASS = 396
const DW_DLE_HIGHPC_WRONG_FORM = 397
const DW_DLE_STR_OFFSETS_BASE_WRONG_FORM = 398
const DW_DLE_DATA16_OUTSIDE_SECTION = 399
const DW_DLE_LNCT_MD5_WRONG_FORM = 400
const DW_DLE_LINE_HEADER_CORRUPT = 401
const DW_DLE_STR_OFFSETS_NULLARGUMENT = 402
const DW_DLE_STR_OFFSETS_NULL_DBG = 403
const DW_DLE_STR_OFFSETS_NO_MAGIC = 404
const DW_DLE_STR_OFFSETS_ARRAY_SIZE = 405
const DW_DLE_STR_OFFSETS_VERSION_WRONG = 406
const DW_DLE_STR_OFFSETS_ARRAY_INDEX_WRONG = 407
const DW_DLE_STR_OFFSETS_EXTRA_BYTES = 408
const DW_DLE_DUP_ATTR_ON_DIE = 409
const DW_DLE_SECTION_NAME_BIG = 410
const DW_DLE_FILE_UNAVAILABLE = 411
const DW_DLE_FILE_WRONG_TYPE = 412
const DW_DLE_SIBLING_OFFSET_WRONG = 413
const DW_DLE_OPEN_FAIL = 414
const DW_DLE_OFFSET_SIZE = 415
const DW_DLE_MACH_O_SEGOFFSET_BAD = 416
const DW_DLE_FILE_OFFSET_BAD = 417
const DW_DLE_SEEK_ERROR = 418
const DW_DLE_READ_ERROR = 419
const DW_DLE_ELF_CLASS_BAD = 420
const DW_DLE_ELF_ENDIAN_BAD = 421
const DW_DLE_ELF_VERSION_BAD = 422
const DW_DLE_FILE_TOO_SMALL = 423
const DW_DLE_PATH_SIZE_TOO_SMALL = 424
const DW_DLE_BAD_TYPE_SIZE = 425
const DW_DLE_PE_SIZE_SMALL = 426
const DW_DLE_PE_OFFSET_BAD = 427
const DW_DLE_PE_STRING_TOO_LONG = 428
const DW_DLE_IMAGE_FILE_UNKNOWN_TYPE = 429
const DW_DLE_LINE_TABLE_LINENO_ERROR = 430
const DW_DLE_PRODUCER_CODE_NOT_AVAILABLE = 431
const DW_DLE_NO_ELF_SUPPORT = 432
const DW_DLE_NO_STREAM_RELOC_SUPPORT = 433
const DW_DLE_RETURN_EMPTY_PUBNAMES_ERROR = 434
const DW_DLE_SECTION_SIZE_ERROR = 435
const DW_DLE_INTERNAL_NULL_POINTER = 436
const DW_DLE_SECTION_STRING_OFFSET_BAD = 437
const DW_DLE_SECTION_INDEX_BAD = 438
const DW_DLE_INTEGER_TOO_SMALL = 439
const DW_DLE_ELF_SECTION_LINK_ERROR = 440
const DW_DLE_ELF_SECTION_GROUP_ERROR = 441
const DW_DLE_ELF_SECTION_COUNT_MISMATCH = 442
const DW_DLE_ELF_STRING_SECTION_MISSING = 443
const DW_DLE_SEEK_OFF_END = 444
const DW_DLE_READ_OFF_END = 445
const DW_DLE_ELF_SECTION_ERROR = 446
const DW_DLE_ELF_STRING_SECTION_ERROR = 447
const DW_DLE_MIXING_SPLIT_DWARF_VERSIONS = 448
const DW_DLE_TAG_CORRUPT = 449
const DW_DLE_FORM_CORRUPT = 450
const DW_DLE_ATTR_CORRUPT = 451
const DW_DLE_ABBREV_ATTR_DUPLICATION = 452
const DW_DLE_DWP_SIGNATURE_MISMATCH = 453
const DW_DLE_CU_UT_TYPE_VALUE = 454
const DW_DLE_DUPLICATE_GNU_DEBUGLINK = 455
const DW_DLE_CORRUPT_GNU_DEBUGLINK = 456
const DW_DLE_CORRUPT_NOTE_GNU_DEBUGID = 457
const DW_DLE_CORRUPT_GNU_DEBUGID_SIZE = 458
const DW_DLE_CORRUPT_GNU_DEBUGID_STRING = 459
const DW_DLE_HEX_STRING_ERROR = 460
const DW_DLE_DECIMAL_STRING_ERROR = 461
const DW_DLE_PRO_INIT_EXTRAS_UNKNOWN = 462
const DW_DLE_PRO_INIT_EXTRAS_ERR = 463
const DW_DLE_NULL_ARGS_DWARF_ADD_PATH = 464
const DW_DLE_DWARF_INIT_DBG_NULL = 465

/* LAST MUST EQUAL LAST ERROR NUMBER */
const DW_DLE_LAST = 465

const DW_DLE_LO_USER = 0x10000

/*  Taken as meaning 'undefined value', this is not
    a column or register number.
    Only present at libdwarf runtime. Never on disk.
    DW_FRAME_* Values present on disk are in dwarf.h
*/
const DW_FRAME_UNDEFINED_VAL = 1034

/*  Taken as meaning 'same value' as caller had, not a column
    or register number
    Only present at libdwarf runtime. Never on disk.
    DW_FRAME_* Values present on disk are in dwarf.h
*/
const DW_FRAME_SAME_VAL = 1035

/* error return values
 */
// const DW_DLV_BADADDR = 0 // (~(Addr)0) // TODO compiler
// const DW_DLV_BADADDR = C.DW_DLV_BADADDR // TODO compiler
const DW_DLV_BADADDR = -1

/* for functions returning target address */

const DW_DLV_NOCOUNT = -1 // ((Signed)(0)-1)
/* for functions returning count */

// const DW_DLV_BADOFFSET = 0 // (~(Dwarf_Off)0)
// const DW_DLV_BADOFFSET = C.DW_DLV_BADOFFSET // TODO compiler
const DW_DLV_BADOFFSET = -1

/* for functions returning offset */

/* standard return values for functions */
const DW_DLV_NO_ENTRY = -1
const DW_DLV_OK = 0
const DW_DLV_ERROR = 1

/* Special values for offset_into_exception_table field of dwarf fde's. */
/* The following value indicates that there is no Exception table offset
   associated with a dwarf frame. */
const DW_DLX_NO_EH_OFFSET = -1 // (-1LL)
/* The following value indicates that the producer was unable to analyse the
   source file to generate Exception tables for this function. */
const DW_DLX_EH_OFFSET_UNAVAILABLE = -2 //(-2LL)

type DwarfFormClass int

const ( // enum Dwarf_Form_Class {
	DW_FORM_CLASS_UNKNOWN = iota
	DW_FORM_CLASS_ADDRESS
	DW_FORM_CLASS_BLOCK
	DW_FORM_CLASS_CONSTANT
	DW_FORM_CLASS_EXPRLOC
	DW_FORM_CLASS_FLAG
	DW_FORM_CLASS_LINEPTR
	DW_FORM_CLASS_LOCLISTPTR   /* DWARF234 only */
	DW_FORM_CLASS_MACPTR       /* DWARF234 only */
	DW_FORM_CLASS_RANGELISTPTR /* DWARF234 only */
	DW_FORM_CLASS_REFERENCE
	DW_FORM_CLASS_STRING
	DW_FORM_CLASS_FRAMEPTR      /* MIPS/IRIX DWARF2 only */
	DW_FORM_CLASS_MACROPTR      /* DWARF5 */
	DW_FORM_CLASS_ADDRPTR       /* DWARF5 */
	DW_FORM_CLASS_LOCLIST       /* DWARF5 */
	DW_FORM_CLASS_LOCLISTSPTR   /* DWARF5 */
	DW_FORM_CLASS_RNGLIST       /* DWARF5 */
	DW_FORM_CLASS_RNGLISTSPTR   /* DWARF5 */
	DW_FORM_CLASS_STROFFSETSPTR /* DWARF5 */
)

/*  These support opening DWARF5 split dwarf objects. */
const DW_GROUPNUMBER_ANY = 0
const DW_GROUPNUMBER_BASE = 1
const DW_GROUPNUMBER_DWO = 2

/*===========================================================================*/
/*  Dwarf consumer interface initialization and termination operations */

/*  Initialization based on path. This is new October 2018.
    The path actually used is copied to true_path_out
    and in the case of MacOS dSYM may not match path.
    So consider the value put in true_path_out the
    actual file name. reserved1,2,3 should all be passed
    as zero. */
func init_path(path string) (
	true_path string, dbg Debug, dwerr Error) {
	ret := C.dwarf_init_path(path.cstr(), nil, 0,
		DW_DLC_READ, 0, 0, 0, &dbg, nil, 0, 0, &dwerr)
	dwerr = packerror(ret, dwerr)
	dbgobj = dbg
	return
}

/*  Initialization based on Unix(etc) open fd */
/*  New March 2017 */
func init_b(fd int) (dbg Debug, dwerr Error) {
	ret := C.dwarf_init_b(fd, DW_DLC_READ, 0, 0, 0, &dbg, &dwerr)
	dwerr = packerror(ret, dwerr)
	dbgobj = dbg
	return
}

func init_a(fd int) (dbg Debug, dwerr Error) {
	ret := C.dwarf_init(fd, DW_DLC_READ, 0, 0, &dbg, &dwerr)
	dwerr = packerror(ret, dwerr)
	dbgobj = dbg
	return
}

func add_file_path(dbg Debug, filename string) (dwerr Error) {
	var dwerr Error
	rv := C.dwarf_add_file_path(dbg, filename.cstr(), &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

/* Undocumented function for memory allocator. */

func print_memory_stats(dbg Debug) {
	C.dwarf_print_memory_stats(dbg)
}

func get_elf(dbg Debug) (
	return_elfptr voidptr, dwerr Error) {
	rv := C.dwarf_get_elf(dbg, &return_elfptr, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}
func finish(dbg Debug) (dwerr Error) {
	rv := C.dwarf_finish(dbg, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

func object_finish(dbg Debug) (dwerr Error) {
	rv := C.dwarf_object_finish(dbg, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

func package_version() string {
	rv := C.dwarf_package_version()
	return gostring(rv)
}

type CUHeader4 struct {
	Length    Unsigned
	Verstamp  Half
	Abbrevoff Off
	Addrsize  Half
	Lensize   Half
	Extsize   Half
	Tysig     Sig8
	Tyoffset  Unsigned
	NextOff   Unsigned
	Type      Half

	CUdie Die
	Index int
}

func next_cu_header4(dbg Debug) (cuhdr *CUHeader4, dwerr Error) {
	cuhdr = &CUHeader4{}
	ret := C.dwarf_next_cu_header_d(dbg, true,
		&cuhdr.Length, &cuhdr.Verstamp, &cuhdr.Abbrevoff,
		&cuhdr.Addrsize, &cuhdr.Lensize, &cuhdr.Extsize,
		&cuhdr.Tysig, &cuhdr.Tyoffset,
		&cuhdr.NextOff, &cuhdr.Type, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

type CUHeader struct {
	Length    Unsigned
	Verstamp  Half
	Abbrevoff Off
	Addrsize  Half
	NextOff   Unsigned
}

func next_cu_header(dbg Debug) (cuhdr *CUHeader, dwerr Error) {
	cuhdr = &CUHeader{}
	ret := C.dwarf_next_cu_header(dbg, &cuhdr.Length, &cuhdr.Verstamp,
		&cuhdr.Abbrevoff, &cuhdr.Addrsize, &cuhdr.NextOff, &dwerr)
	dwerr = packerror(ret, dwerr)
	return cuhdr
}

func siblingof2(dbg Debug, die Die) (siblingdie Die, dwerr Error) {
	isinfo := 1
	ret := C.dwarf_siblingof_b(dbg, die, isinfo, &siblingdie, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}
func siblingof(dbg Debug, die Die) (siblingdie Die, dwerr Error) {
	ret := C.dwarf_siblingof(dbg, die, &siblingdie, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func dealloc(dbg Debug, space voidptr, type_ int) {
	C.dwarf_dealloc(dbg, space, type_) // DW_DLA_DIE ..
}

/* New 27 April 2015. */
func die_from_hash_signature(dbg Debug, hash_sig *Sig8, sig_type string) (
	ret_CU_die Die, dwerr Error) {
	ret := C.dwarf_die_from_hash_signature(dbg, hash_sig, sig_type.cstr(),
		&ret_CU_die, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func child(die Die) (childdie Die, dwerr Error) {
	ret := C.dwarf_child(die, &childdie, &dwerr)
	dwerr = packerror(ret, dwerr)
	if false {
		(&dwerr).with(ret) // ok but syntax not good
	}
	return
}

/*  Section name access.  Because sections might
    now end with .dwo or be .zdebug  or might not.
*/
func get_die_section_name(dbg Debug, isinfo bool) (sec_name string, dwerr Error) {
	var sec_namep byteptr
	rv := C.dwarf_get_die_section_name(dbg, isinfo, &sec_namep, &dwerr)
	dwerr = packerror(rv, dwerr)
	sec_name = gostring(sec_namep)
	return
}
func get_die_section_name_b(die Die) (sec_name string, dwerr Error) {
	var sec_namep byteptr
	rv := C.dwarf_get_die_section_name_b(die, &sec_namep, &dwerr)
	dwerr = packerror(rv, dwerr)
	sec_name = gostring(sec_namep)
	return
}

func attr(die Die, attr Half) (ret_attr Attribute, dwerr Error) {
	ret := C.dwarf_attr(die, attr, &ret_attr, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}
func die_text(die Die, attr Half) (retname string, dwerr Error) {
	var retnamep byteptr
	ret := C.dwarf_die_text(die, attr, &retnamep, &dwerr)
	dwerr = packerror(ret, dwerr)
	retname = gostring(retnamep)
	return
}

func diename(die Die) (diename string, dwerr Error) {
	var dienamep byteptr
	ret := C.dwarf_diename(die, &dienamep, &dwerr)
	dwerr = packerror(ret, dwerr)
	diename = gostring(dienamep)
	// dwarf_dealloc(dbg Debug, dienamep, DW_DLA_STRING)
	return
}

/* convenience functions, alternative to using dwarf_attrlist */
func hasattr(die Die, attr Half) (ret_bool Bool, dwerr Error) {
	ret := C.dwarf_hasattr(die, attr, &ret_bool, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

type AttrList struct {
	Atbuf *Attribute
	Atcnt Signed
}

func (die Die) GetAttrs() {

}

func attrlist(die Die) (attrbuf *Attribute, attrcnt Signed, dwerr Error) {
	ret := C.dwarf_attrlist(die, &attrbuf, &attrcnt, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

// func tag(die Die) (tag dwarf)

func (die Die) Lowpc() {

}
func lowpc(die Die) (retaddr Addr, dwerr Error) {
	ret := C.dwarf_lowpc(die, &retaddr, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

/*  This works for DWARF2 and DWARF3 styles of DW_AT_highpc,
    but not for the DWARF4 class constant forms.
    If the FORM is of class constant this returns an error */
func (die Die) Highpc() {

}
func highpcx(die Die) (retaddr Addr, dwerr Error) {
	hival, form, class, dwerr2 := highpc2(die)
	dwerr = dwerr2
	if dwerr.Okay() {
		if class == DW_FORM_CLASS_CONSTANT {
			lowaddr, dwerr3 := lowpc(die)
			dwerr = dwerr3
			if dwerr3.Okay() {
				retaddr = lowaddr + hival
			}
		} else {
			retaddr = hival
		}
	}
	return
}

func highpc2(die Die) (value Addr, form Half, class int, dwerr Error) {
	ret := C.dwarf_highpc_b(die, &value, &form, &class, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}
func highpc(die Die) (retaddr Addr, dwerr Error) {
	ret := C.dwarf_highpc(die, &retaddr, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func (attr Attribute) Whatattr() {
}
func whatattr(attr Attribute) (ret_attr_num Half, dwerr Error) {
	ret := C.dwarf_whatattr(attr, &ret_attr_num, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func (attr Attribute) Hasform() {
}
func (attr Attribute) Whatform() {
}
func whatform(attr Attribute) (ret_form_num Half, dwerr Error) {
	ret := C.dwarf_whatform(attr, &ret_form_num, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func global_formref(attr Attribute) (offset Off, dwerr Error) {
	ret := C.dwarf_global_formref(attr, &offset, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func formaddr(attr Attribute) (addr Addr, dwerr Error) {
	C.dwarf_formaddr(attr, &addr, &dwerr)
	return
}

func formudata(attr Attribute) (val Unsigned, dwerr Error) {
	ret := C.dwarf_formudata(attr, &val, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func formstring(attr Attribute) (str string, dwerr Error) {
	var strp byteptr
	ret := C.dwarf_formstring(attr, &strp, &dwerr)
	dwerr = packerror(ret, dwerr)
	str = gostring(strp)
	return
}

func srcfiles(die Die) (files []string, dwerr Error) {
	var srcfilesp *byteptr
	var filecount Signed
	ret := C.dwarf_srcfiles(die, &srcfilesp, &filecount, &dwerr)
	dwerr = packerror(ret, dwerr)
	files2 := []string{}
	for i := 0; i < filecount; i++ {
		// files = append(files, "file"+i.repr()) // TODO compiler
		filename := "file" + i.repr()
		filename = gostring_clone(srcfilesp[i])
		files2 = append(files2, filename)
	}
	files = files2

	// clean
	dbg := dbgobj
	for i := 0; i < filecount; i++ {
		dealloc(dbg, srcfilesp[i], DW_DLA_STRING)
	}
	dealloc(dbg, srcfilesp, DW_DLA_LIST)
	return
}

/* Start line number operations */
/* dwarf_srclines  is the original interface from 1993. */
func srclines(die Die) (linebuf *Line, linecount Signed, dwerr Error) {
	ret := C.dwarf_srclines(die, &linebuf, &linecount, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func srclines_dealloc(dbg Debug, linebuf *Line, count Signed) {
	C.dwarf_srclines_dealloc(dbg, linebuf, count)
}

func srclines2(die Die) (version_out Unsigned,
	table_cout Small, linecontext LineContext, dwerr Error) {
	ret := C.dwarf_srclines_b(die, &version_out, &table_cout, &linecontext, &dwerr)
	dwerr = packerror(ret, dwerr)
	return
}

func srclines_from_linecontext(lctx LineContext) (
	linebuf *Line, linecount Signed, dwerr Error) {
	rv := C.dwarf_srclines_from_linecontext(lctx, &linebuf, &linecount, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

func lineaddr(line Line) (retaddr Addr, dwerr Error) {
	rv := C.dwarf_lineaddr(line, &retaddr, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

func lineendsequence(line Line) (retbool Bool, dwerr Error) {
	rv := C.dwarf_lineendsequence(line, &retbool, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

func lineno(line Line) (ret_lineno Unsigned, dwerr Error) {
	rv := C.dwarf_lineno(line, &ret_lineno, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}
func linesrc(line Line) (retname string, dwerr Error) {
	var namep byteptr
	rv := C.dwarf_linesrc(line, &namep, &dwerr)
	dwerr = packerror(rv, dwerr)
	if dwerr.Okay() {
		retname = gostring_clone(namep)
		dealloc(dbgobj, namep, DW_DLA_STRING)
	}
	return
}
func line_srcfileno(line Line) (ret_fileno Unsigned, dwerr Error) {
	rv := C.dwarf_line_srcfileno(line, &ret_fileno, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

/* dwarf_srclines_dealloc_b(), created October 2015, is the
   appropriate method for deallocating everything
   and dwarf_srclines_from_linecontext(),
   dwarf_srclines_twolevel_from_linecontext(),
   and dwarf_srclines_b()  allocate.  */
func srclines_dealloc2(line_context LineContext) {
	C.dwarf_srclines_dealloc_b(line_context)
}

func get_ranges1(dbg Debug, offset Off, die Die) (
	rangesbuf *Ranges, listlen Signed, bytecount Unsigned, dwerr Error) {
	rv := C.dwarf_get_ranges_a(dbg, offset, die,
		&rangesbuf, &listlen, &bytecount, &dwerr)
	dwerr = packerror(rv, dwerr)
	return
}

func ranges_dealloc(dbg Debug, rangesbuf *Ranges, rangecount Signed) {
	C.dwarf_ranges_dealloc(dbg, rangesbuf, rangecount)
}
