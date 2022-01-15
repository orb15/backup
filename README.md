# backup
Backup is a golang-based quick and dirty tool to back up local disks to AWS S3. I wrote it after being dissappointed with the performance and stability of high-volume transfers to a bucket via the AWS console

# Features
* regex-based rules to exclude directories and files expressed in an external config file
* skips folders that begin with '.' for security reasons (see below)
* creates one bucket per archive with an S3 'folder structure' that mimics the archived files
* an external file to define which folders to back up
* file transfer validation via MD5 hash comparison
* a dryrun mode
* high thruput and performance (relative to AWS Console transfers at least)
* file transfer retry with an exponential backoff
* uniform json logging for log post-processing
* automated bucket naming with date and uuid to prevent bucket name conflicts
* multithreaded file hashing and networking
* files that failed transfer after retry are listed in a JSON file for subsequent re-uploading

# Performance
Better than AWS console? Ehh, I am seeing 25K files / 19GB from 1 spinning disk hashed and sent to S3 in 2h30m. 99.98% of all transfers were successful  
&nbsp;&nbsp;&nbsp;&nbsp;Test Machine Specs: 4phys/8virtual 3.5Gz CPU with 16GB RAM. HDD is WD Black 7200RPM 1TB. File fragmentation is low to moderate  
&nbsp;&nbsp;&nbsp;&nbsp;Network Bandwidth: 30Mbps sustained during transfer  
&nbsp;&nbsp;&nbsp;&nbsp;During hashing: HDD is the performance limiter with plenty of room on CPU and RAM  
&nbsp;&nbsp;&nbsp;&nbsp;During transfer: network bandwidth is the performance limiter with almost no CPU, RAM or Disk access

# Limitations and Improvements
* developed on Windows since that is where I needed it. There are a couple of places with Windows-isms that need to be addressed
* need to extend command line options for a dozen or so params. Right now most config defaults to what I needed
* probably need to allow users to specify bucket attributes beyond the very basic ones hardcoded in the system (applying ACLs, setting lifecycle stuff etc)
* need to tune the multithreading parameters to optimize for workload
* may want to move file hashing to occur just before file transfer - might improve efficiency since file isn't opened and closed twice - once to hash and then again to transfer. May have unexpectedly bad impact on transfer performance though given that md5 hashing performance is already disk bound. More work is needed here

# Security
* relies on external AWS credentials file stored in the usual location(s). See AWS docs for how to configure AWS for secure command line operations
* <span style="color:red">never place your AWS credentials in a folder that will be pushed to AWS or GitHub!</span>
* <span style="color:red">be careful you do not accidently add your AWS creds to backup! This code ignores folders that begin with '.', which should protect you if you are following standard AWS guidelines, but be certain you know what you are sending to the cloud before backing anything up!</span>
* <span style="color:red">when using AWS command line tools, ALWAYS use an IAM account with minimal privliges. This code requires S3 List, PutObject and Bucket Creation rights. It does NOT download from S3 (no reading rights needed) nor does it ever delete data (no deletion rights needed). It does NOT need any other access, so use an IAM with as limited a security footprint as possible</span>