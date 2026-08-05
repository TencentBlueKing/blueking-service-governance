package bkerrs

import "fmt"

// WrapBSCPNoPermission wraps a BSCP permission error into the standard API shape.
func WrapBSCPNoPermission(err error, msg string) *Error {
	return Wrap(err, ErrCodeNoPermission, msg).SetDetails(
		NewDetail(
			ErrDetailCodeBSCPNoPermission,
			"bscp service permission denied",
			WithSystem(SystemName),
			WithModule("bscp"),
		),
	)
}

// WrapBSCPNotFullyReleased 包装为 BSCP 服务未全量发布错误
func WrapBSCPNotFullyReleased(err error, bizID, serviceID string) error {
	wrappedErr := Wrapf(err, ErrCodeNotFound, "no fully released version for service: %s in biz: %s", serviceID, bizID)
	return wrappedErr.SetDetails(
		NewDetail(
			ErrDetailCodeNotFullyReleased,
			fmt.Sprintf("bscp service %s has no fully released version in biz %s", serviceID, bizID),
			WithSystem("bkms"),
			WithModule("bscp"),
		),
	)
}
