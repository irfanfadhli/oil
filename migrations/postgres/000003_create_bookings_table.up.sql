CREATE TABLE room_bookings (
  id VARCHAR(36) PRIMARY KEY,

  room_id VARCHAR(36) NOT NULL,

  guest_name VARCHAR(100) NOT NULL,
  guest_email VARCHAR(100),
  guest_phone VARCHAR(20),

  booking_date DATE NOT NULL,
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,

  purpose TEXT,
  status VARCHAR(20) DEFAULT 'pending',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by VARCHAR(36) NOT NULL,
  modified_by VARCHAR(36) NOT NULL,

  CONSTRAINT fk_room
    FOREIGN KEY (room_id)
    REFERENCES rooms(id)
    ON DELETE CASCADE
);

