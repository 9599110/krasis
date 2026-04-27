-- Seed admin user (password: kaixin100)
INSERT INTO users (id, email, username, password_hash, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@krasis.local',
    'admin',
    '$2a$10$M4ZLA4OM2mmG39gp7hlzPOLzrT/uBMevJpBnK1BVz8XwcvYvcRA6S',
    1
) ON CONFLICT DO NOTHING;

-- Assign admin role
INSERT INTO user_roles (user_id, role_id)
SELECT
    '00000000-0000-0000-0000-000000000001',
    id
FROM roles
WHERE name = 'admin'
ON CONFLICT DO NOTHING;
