# backup
Backup is a golang-based quick and dirty tool to back up local disks to AWS S3. I wrote it after being dissappointed with the performance and stability of high-volume transfers to a bucket via the AWS console

# Features
* regex-based rules to exclude directories and files expressed in an external config file
* an external file to define which folders to back up
* AWS file transfer validation
* a dryrun mode
* high throuput and performance
* file transfer retry with an exponential backoff
* uniform json logging for log post-processing
* automated bucket naming with date and uuid to prevent bucket name conflicts
* multithreaded file hashing and networking

# Performance
Better than AWS console? Ehh, I am seeing 25K files / 19GB from 1 spinning disk hashed and sent to S3 in 2h30m on a 4phys/8virtual 3.5Gz CPU with 16GB RAM with 30Mbps sustained bandwidth while hitting 99.98% successful transfers

# Limitations
* developed on windows since that is where I needed it so there are a couple of places with Windows-isms that need to be addressed
* need to extend command line options for a dozen or so params. Right now most config defaults to what I needed
* probably should write all failures to a user-specified file for later post-processing or even use this file as the basis for a targeted re-upload to complete an archive
* probably need to allow users to specify bucket attributes beyond the very basic ones hardcoded in the system (applying ACLs, setting lifecycle stuff etc)
