# Transferring Credentials to Another Computer

__Difficulty__: Advanced.

At times you may need to transfer the credentials to another computer or
system, such as CI/CD.  As the credentials are encrypted with the machine-specific
key (machine ID), the credentials will not work straight away.

Slackdump supports Machine ID override, which allows you to define the machine ID
for a chosen workspace.  Using the same Machine ID on your local and remote systems
will allow you to use the same credentials on both.

__IMPORTANT__:  Never share your custom machine ID with anyone.  The machine ID
is a secret key to your login information.

To transfer the credentials to another system, follow these steps:

1. Reauthenticate in the workspace you want to transfer by specifying the
   machine ID override with `-machine-id` flag. For example: 
    
   ```bash
   slackdump workspace new -machine-id="my-machine-id" your_workspace
   ```

   This will create a new workspace file with the machine ID override.

2. Run the `slackdump workspace list` command to get the workspace file name,
   for example: `your_workspace.bin`

3. Find out the slackdump cache directory location on your system by running:

   ```bash
   slackdump tools info
   ```

   Slackdump cache location will be in "workspace" section, "path" field.  If
   you have `jq` installed, you can run:

   ```bash
   slackdump tools info | jq -r '.workspace.path'
   ```

5. Install slackdump on the remote system.

6. Repeat the step 3 on the remote system to find out the cache directory.

7. Create it if it doesn't exist

8. Copy the workspace file and `workspace.txt` file from the cache directory to
   the remote system.

9. Verify that the workspace is available and credentials are working by running:

   ```bash
   slackdump workspace list -a -machine-id="my-machine-id" 
   ```

   on the remote system.
   You should see OK in the last "error" column.  If you see "failed to load
   stored credentials", it means that the credentials are not working.

10. You can now use the credentials on the remote system.