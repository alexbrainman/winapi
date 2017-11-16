// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package winapi

const (
	ACL_REVISION = 2

	SE_UNKNOWN_OBJECT_TYPE     = 0
	SE_FILE_OBJECT             = 1
	SE_SERVICE                 = 2
	SE_PRINTER                 = 3
	SE_REGISTRY_KEY            = 4
	SE_LMSHARE                 = 5
	SE_KERNEL_OBJECT           = 6
	SE_WINDOW_OBJECT           = 7
	SE_DS_OBJECT               = 8
	SE_DS_OBJECT_ALL           = 9
	SE_PROVIDER_DEFINED_OBJECT = 10
	SE_WMIGUID_OBJECT          = 11
	SE_REGISTRY_WOW64_32KE     = 12

	OWNER_SECURITY_INFORMATION            = 0x00000001
	GROUP_SECURITY_INFORMATION            = 0x00000002
	DACL_SECURITY_INFORMATION             = 0x00000004
	SACL_SECURITY_INFORMATION             = 0x00000008
	LABEL_SECURITY_INFORMATION            = 0x00000010
	UNPROTECTED_SACL_SECURITY_INFORMATION = 0x10000000
	UNPROTECTED_DACL_SECURITY_INFORMATION = 0x20000000
	PROTECTED_SACL_SECURITY_INFORMATION   = 0x40000000
	PROTECTED_DACL_SECURITY_INFORMATION   = 0x80000000
)

type ACL struct {
	AclRevision byte
	Sbz1        byte
	AclSize     uint16
	AceCount    uint16
	Sbz2        uint16
}

type SECURITY_INFORMATION uint32

//sys	InitializeAcl(acl *ACL, acllen uint32, aclrev uint32) (err error) = advapi32.InitializeAcl
//sys	SetNamedSecurityInfo(objname *uint16, objtype int32, secinfo SECURITY_INFORMATION, owner *syscall.SID, group *syscall.SID, dacl *ACL, sacl *ACL) (errno uint32) = advapi32.SetNamedSecurityInfoW
