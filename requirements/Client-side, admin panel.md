# Overview

The admin panel must including the following:
- `/login` - Login, redirect here if invalid session cookie or no cookie
- `/tracking` - Showing a table of all admin trackings, and a sort/filter table.  Initially shows nothing until requested to.
- `/tracking/$trackingId` - Shows details of  a tracking, including all of its statuses. Need a status updating section.

If the user role is `admin`, add the following:
- `/accounts` - For creating and deleting accounts.