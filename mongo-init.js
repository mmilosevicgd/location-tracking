let usersToCreate = [
    {
        user: "location-management-service",
        pwd: "location-management-service-password",
        roles: [
            {
                role: "readWrite",
                db: "location-management-db"
            }
        ]
    },
    {
        user: "location-history-management-service",
        pwd: "location-history-management-service-password",
        roles: [
            {
                role: "readWrite",
                db: "location-history-management-db"
            }
        ]
    }
];

let adminDb = db.getSiblingDB("admin");

usersToCreate.forEach(function(user) {
    adminDb.createUser(user);
});