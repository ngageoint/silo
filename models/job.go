package models

import (
	"database/sql"
	"log"
	"strings"

	"github.com/ngageoint/seed-common/util"
)

type Job struct {
	ID                   int    `db:"id"`
	Name                 string `db:"name"`
	LatestJobVersion     string `db:"latest_job_version"`
	LatestPackageVersion string `db:"latest_package_version"`
	Title                string `db:"title"`
	Maintainer           string `db:"maintainer"`
	Email                string `db:"email"`
	MaintOrg             string `db:"maint_org"`
	Description          string `db:"description"`
	ImageIDs             []int
	JobVersions          []JobVersion
}

func SetJobInfo(job *Job, img Image) {
	job.Name = img.ShortName
	job.LatestJobVersion = img.JobVersion
	job.LatestPackageVersion = img.PackageVersion
	job.Title = img.Seed.Job.Title
	job.Maintainer = img.Maintainer
	job.Email = img.Email
	job.MaintOrg = img.MaintOrg
	job.Description = img.Description
}

func CreateJobTable(db *sql.DB, dbType string) {
	// create table if it does not exist
	sql_table := `
	CREATE TABLE IF NOT EXISTS Job(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		latest_job_version TEXT,
		latest_package_version TEXT,
		title TEXT,
		maintainer TEXT,
		email TEXT,
		maint_org TEXT,
		description TEXT
	);
	`
	if dbType == "postgres" {
	    sql_table = strings.Replace(sql_table, "id INTEGER PRIMARY KEY AUTOINCREMENT", "id SERIAL PRIMARY KEY", 1)
	}

	_, err := db.Exec(sql_table)
	if err != nil {
		panic(err)
	}
}

func ResetJobTable(db *sql.DB, dbType string) error {
    if dbType == "sqlite" {
        return ResetJobTableLite(db)
    } else if dbType == "postgres" {
        return ResetJobTablePG(db)
    } else {
        panic("unsupported database type")
    }
}

func ResetJobTableLite(db *sql.DB) error {
	// delete all jobs and reset the counter
	delete := `DELETE FROM Job;`

	_, err := db.Exec(delete)
	if err != nil {
		panic(err)
	}

	reset := `DELETE FROM sqlite_sequence WHERE NAME='Job';`

	_, err2 := db.Exec(reset)
	if err2 != nil {
		panic(err2)
	}

	return err2
}

func ResetJobTablePG(db *sql.DB) error {
	// delete all images and reset the counter
	delete := `TRUNCATE Job RESTART IDENTITY CASCADE;`

	_, err := db.Exec(delete)
	if err != nil {
		panic(err)
	}

	return err
}

func BuildJobsList(db *sql.DB, images *[]Image, dbType string) []Job {
	jobs := []Job{}
	jobMap := make(map[string]Job)
	jobVersions := []JobVersion{}
	jvMap := make(map[string]JobVersion)
	(*images)[0].JobId = 1
	for i, _ := range *images {
		img := &(*images)[i]
		img.ShortName = img.Seed.Job.Name
		img.JobVersion = img.Seed.Job.JobVersion
		img.PackageVersion = img.Seed.Job.PackageVersion

		versionName := img.ShortName + img.JobVersion

		job, ok := jobMap[img.ShortName]
		if ok {
			jv := img.JobVersion
			pv := img.PackageVersion
			lj := job.LatestJobVersion
			lp := job.LatestPackageVersion
			if jv > lj || (jv == lj && pv > lp) {
				SetJobInfo(&job, *img)
				UpdateJob(db, job)
			}
		}
		if !ok {
			job = Job{}
			SetJobInfo(&job, *img)

			var id int
			var err2 error
			if dbType == "postgres" {
				id, err2 = AddJobPg(db, job)
			} else {
				id, err2 = AddJobLite(db, job)
			}
			if err2 != nil {
				util.PrintUtil("ERROR: Error adding job in BuildJobsList: %v\n", err2)
			}

			job.ID = id
			jobMap[img.ShortName] = job
			jobs = append(jobs, job)
		}

		img.JobId = job.ID

		jobVersion, ok := jvMap[versionName]
		if ok {
			pv := img.PackageVersion
			lp := jobVersion.LatestPackageVersion
			if pv > lp {
				SetJobVersionInfo(&jobVersion, *img)
				UpdateJobVersion(db, jobVersion)
			}
		}
		if !ok {
			jobVersion = JobVersion{}
			SetJobVersionInfo(&jobVersion, *img)

			var id int
			var err2 error
			if dbType == "postgres" {
				id, err2 = AddJobVersionPg(db, jobVersion)
			} else {
				id, err2 = AddJobVersionLite(db, jobVersion)
			}
			if err2 != nil {
				util.PrintUtil("ERROR: Error adding job version in BuildJobsList: %v\n", err2)
			}

			jobVersion.ID = id
			jvMap[versionName] = jobVersion
			jobVersions = append(jobVersions, jobVersion)
		}

		img.JobVersionId = jobVersion.ID

	}

	return jobs
}

func AddJobLite(db *sql.DB, job Job) (int, error) {
	sql_add := `
	INSERT INTO Job(
		name,
		latest_job_version,
		latest_package_version,
		title,
		maintainer,
		email,
		maint_org,
		description
	) values($1, $2, $3, $4, $5, $6, $7, $8)
	`

	stmt, err := db.Prepare(sql_add)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(job.Name, job.LatestJobVersion, job.LatestPackageVersion,
		job.Title, job.Maintainer, job.Email, job.MaintOrg, job.Description)
	if err != nil {
		return -1, err
	}

	id := -1
	var id64 int64
	if err == nil {
		id64, err = result.LastInsertId()
		id = int(id64)
	}

	return id, err
}

func AddJobPg(db *sql.DB, job Job) (int, error) {
	query :=
	`INSERT INTO Job(
			name, 
			latest_job_version,
			latest_package_version,
			title,
			maintainer,
			email,
			maint_org,
			description) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;`


	var id int
	err := db.QueryRow(query, job.Name, job.LatestJobVersion, job.LatestPackageVersion,
		job.Title, job.Maintainer, job.Email, job.MaintOrg, job.Description).Scan(&id)

	return id, err
}

func UpdateJob(db *sql.DB, job Job) error {
	sql_update := `UPDATE Job SET 
		latest_job_version=$1, 
		latest_package_version=$2,		
		title=$3,
		maintainer=$4,
		email=$5,
		maint_org=$6,
		description=$7
		where id=$8`


	stmt, err := db.Prepare(sql_update)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(job.LatestJobVersion, job.LatestPackageVersion,
		job.Title, job.Maintainer, job.Email, job.MaintOrg, job.Description, job.ID)

	return err
}

func UpdateJobPg(db *sql.DB, job Job) error {
	sql_update := `UPDATE Job SET 
		latest_job_version=$1, 
		latest_package_version=$2,		
		title=$3,
		maintainer=$4,
		email=$5,
		maint_org=$6,
		description=$7
		where id=$8`

	stmt, err := db.Prepare(sql_update)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(job.LatestJobVersion, job.LatestPackageVersion,
		job.Title, job.Maintainer, job.Email, job.MaintOrg, job.Description, job.ID)

	return err
}

func StoreJobs(db *sql.DB, jobs []Job) {
	sql_add := `
	INSERT INTO Job(
		name,
		latest_job_version,
		latest_package_version,
		title,
		maintainer,
		email,
		description
	) values(?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_add)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	for _, job := range jobs {
		_, err2 := stmt.Exec(job.Name, job.LatestJobVersion, job.LatestPackageVersion,
			job.Title, job.Maintainer, job.Email, job.MaintOrg, job.Description)
		if err2 != nil {
			panic(err2)
		}
	}
}

func ReadJobs(db *sql.DB) []Job {
	sql_readall := `
	SELECT * FROM Job
	ORDER BY id ASC
	`

	rows, err := db.Query(sql_readall)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []Job
	for rows.Next() {
		item := Job{}
		err2 := rows.Scan(&item.ID, &item.Name, &item.LatestJobVersion, &item.LatestPackageVersion,
			&item.Title, &item.Maintainer, &item.Email, &item.MaintOrg, &item.Description)
		if err2 != nil {
			panic(err2)
		}

		item.ImageIDs = GetJobImageIds(db, item.ID)
		item.JobVersions = GetJobVersions(db, item.ID)

		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		util.PrintUtil("ERROR: Error in ReadJobs: %v", err)
	}
	return result
}

func ReadJob(db *sql.DB, id int) (Job, error) {
	row := db.QueryRow("SELECT * FROM Job WHERE id=$1", id)

	var result Job
	err := row.Scan(&result.ID, &result.Name, &result.LatestJobVersion, &result.LatestPackageVersion,
		&result.Title, &result.Maintainer, &result.Email, &result.MaintOrg, &result.Description)
	if err != nil {
		util.PrintUtil("ERROR scanning in read job: %v", err.Error())
	}

	return result, err
}

type JobVersion struct {
	ID                   int    `db:"id"`
	JobName              string `db:"job_name"`
	JobId                int    `db:"job_id"`
	JobVersion           string `db:"job_version"`
	LatestPackageVersion string `db:"latest_package_version"`
	Images               []SimpleImage
}

func SetJobVersionInfo(jv *JobVersion, img Image) {
	jv.JobName = img.ShortName
	jv.JobId = img.JobId
	jv.JobVersion = img.JobVersion
	jv.LatestPackageVersion = img.PackageVersion
}

func CreateJobVersionTable(db *sql.DB, dbType string) {
	// create table if it does not exist
	sql_table := `
	CREATE TABLE IF NOT EXISTS JobVersion(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_name TEXT,
		job_id INTEGER NOT NULL,
		job_version TEXT,
		latest_package_version TEXT,
		UNIQUE(job_name, job_version),
		CONSTRAINT fk_inv_job_id
		    FOREIGN KEY (job_id)
		    REFERENCES Job (id)
		    ON DELETE CASCADE
	);
	`

	if dbType == "postgres" {
        sql_table = strings.Replace(sql_table, "id INTEGER PRIMARY KEY AUTOINCREMENT", "id SERIAL PRIMARY KEY", 1)
    }

	_, err := db.Exec(sql_table)
	if err != nil {
		panic(err)
	}
}

func ResetJobVersionTable(db *sql.DB, dbType string) error {
    if dbType == "sqlite" {
        return ResetJobVersionTableLite(db)
    } else if dbType == "postgres" {
        return ResetJobVersionTablePG(db)
    } else {
        panic("unsupported database type")
    }
}

func ResetJobVersionTableLite(db *sql.DB) error {
	// delete all job versions and reset the counter
	delete := `DELETE FROM JobVersion;`

	_, err := db.Exec(delete)
	if err != nil {
		panic(err)
	}

	reset := `DELETE FROM sqlite_sequence WHERE NAME='JobVersion';`

	_, err2 := db.Exec(reset)
	if err2 != nil {
		panic(err2)
	}

	return err2
}

func ResetJobVersionTablePG(db *sql.DB) error {
	// delete all images and reset the counter
	delete := `TRUNCATE JobVersion RESTART IDENTITY CASCADE;`

	_, err := db.Exec(delete)
	if err != nil {
		panic(err)
	}

	return err
}

func AddJobVersionLite(db *sql.DB, jv JobVersion) (int, error) {
	sql_add := `
	INSERT INTO JobVersion(
		job_name,
		job_id,
		job_version,
		latest_package_version
	) values(?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_add)
	if err != nil {
		return -1, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(jv.JobName, jv.JobId, jv.JobVersion, jv.LatestPackageVersion)
	if err != nil {
		return -1, err
	}

	id := -1
	var id64 int64
	if err == nil {
		id64, err = result.LastInsertId()
		id = int(id64)
	}

	return id, err
}

func AddJobVersionPg(db *sql.DB, jv JobVersion) (int, error) {
	query :=
		`INSERT INTO JobVersion(
			job_name,
			job_id,
			job_version,
			latest_package_version
		)	VALUES($1, $2, $3, $4) RETURNING id;`


	var id int
	err := db.QueryRow(query, jv.JobName, jv.JobId, jv.JobVersion,
		jv.LatestPackageVersion).Scan(&id)

	return id, err
}

func UpdateJobVersion(db *sql.DB, jv JobVersion) error {
	sql_update := `UPDATE JobVersion SET 
		job_name=$1,
		job_id=$2,
		job_version=$3,
		latest_package_version=$4
		where id=$5`

	stmt, err := db.Prepare(sql_update)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(jv.JobName, jv.JobId, jv.JobVersion, jv.LatestPackageVersion, jv.ID)

	return err
}

func ReadJobVersions(db *sql.DB) []JobVersion {
	sql_read := `
	SELECT * FROM JobVersion
	ORDER BY id ASC
	`

	rows, err := db.Query(sql_read)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []JobVersion
	for rows.Next() {
		item := JobVersion{}
		err2 := rows.Scan(&item.ID, &item.JobName, &item.JobId, &item.JobVersion, &item.LatestPackageVersion)
		if err2 != nil {
			panic(err2)
		}

		item.Images = GetJobVersionImages(db, item.ID)

		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		util.PrintUtil("ERROR: Error in ReadJobVersions: %v", err)
	}
	return result
}

func ReadJobVersion(db *sql.DB, id int) (JobVersion, error) {
	row := db.QueryRow("SELECT * FROM JobVersion WHERE id=$1", id)

	var result JobVersion
	err := row.Scan(&result.ID, &result.JobName, &result.JobId, &result.JobVersion, &result.LatestPackageVersion)
	if err != nil {
		util.PrintUtil("ERROR scanning in read job version: %v", err.Error())
	}

	result.Images = GetJobVersionImages(db, result.ID)

	return result, err
}

func GetJobVersions(db *sql.DB, jobid int) []JobVersion {
	sql_readall := `SELECT * FROM JobVersion WHERE job_id=$1`

	rows, err := db.Query(sql_readall, jobid)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var result []JobVersion
	for rows.Next() {
		item := JobVersion{}
		err2 := rows.Scan(&item.ID, &item.JobName, &item.JobId, &item.JobVersion, &item.LatestPackageVersion)
		if err2 != nil {
			panic(err2)
		}

		item.Images = GetJobVersionImages(db, item.ID)

		result = append(result, item)
	}

	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return result
}
