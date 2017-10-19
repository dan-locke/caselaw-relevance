
CREATE TABLE users (

	user_id SERIAL UNIQUE,

	name VARCHAR(255) NOT NULL,

	pass VARCHAR(20) NOT NULL,

	PRIMARY KEY (name, pass)

);

CREATE TYPE assessType AS ENUM('not relevant', 'background', 'explanatory', 'on point');

CREATE TABLE tag (

	topic_id bigint,

	doc_id bigint,

	tagger int,

	date_added TIMESTAMP,

	start_pos bigint NOT NULL,

	end_pos bigint NOT NULL,

	PRIMARY KEY (topic_id, doc_id, tagger, date_added),

	FOREIGN KEY (tagger) REFERENCES users (user_id)

);

CREATE TABLE assessment (

	assessment_id SERIAL,

	doc_id bigint NOT NULL,

	topic_id bigint NOT NULL,

	assessor int NOT NULL,

	relevant assessType,

	date_assessed TIMESTAMP,

	PRIMARY KEY (assessment_id),

	FOREIGN KEY (assessor) REFERENCES users (user_id)

);

CREATE TABLE query (

		query_id SERIAL,

		user_id int NOT NULL,

		topic_id bigint NOT NULL,

		query text NOT NULL,

		date_added TIMESTAMP,

		PRIMARY KEY (query_id),

		FOREIGN KEY (user_id) REFERENCES users (user_id)

);
