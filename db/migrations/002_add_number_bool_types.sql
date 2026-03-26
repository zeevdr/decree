-- +goose Up

-- Rename "int" to "integer" for OAS consistency before creating the enum.
UPDATE schema_fields SET field_type = 'integer' WHERE field_type = 'int';

-- Create a proper enum type to enforce valid field types at the DB level.
CREATE TYPE field_type AS ENUM (
    'integer',
    'number',
    'string',
    'bool',
    'time',
    'duration',
    'url',
    'json'
);

-- Convert the column from TEXT to the enum.
ALTER TABLE schema_fields
    ALTER COLUMN field_type TYPE field_type USING field_type::field_type;

-- +goose Down

ALTER TABLE schema_fields
    ALTER COLUMN field_type TYPE TEXT USING field_type::TEXT;

DROP TYPE field_type;

UPDATE schema_fields SET field_type = 'int' WHERE field_type = 'integer';
