/** The shortest password an administrator may have, and a user: the
    server's model.MinAdminPasswordLength and MinUserPasswordLength, said here
    too so a form can say so before anything is sent
    (TestThePagesKnowThePasswordLengths holds them to the server's). */
export const MIN_ADMIN_PASSWORD = 14;
export const MIN_USER_PASSWORD = 12;

export const ADMIN_DEPENDENCY = 'app:admin';

/** What the server takes for the organization's logo (model.NewOrganizationLogo), said here too
    so the picker can refuse a file before sending it. The server decides by the bytes. */
export const ORGANIZATION_LOGO = {
	maxBytes: 512 * 1024,
	types: ['image/png', 'image/jpeg'],
	accept: 'image/png,image/jpeg,.png,.jpg,.jpeg'
} as const;
