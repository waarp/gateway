Consulter un certificat
=======================

.. http:get:: /api/partners/(partner)/accounts/(account)/certificates/(certificate)

   Renvoie le certificat `certificate` du compte `account` associé au partenaire
   nommé `partner`.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          GET /api/partners/partenaire1/accounts/utilisateur1/certificates/certificat1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le certificat a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat, compte ou partenaire demandé n'existe pas

   :Response JSON Object:

       * **Name** (*string*) - Le nom du certificat
       * **PrivateKey** (*string*) - La clé privée du certificat
       * **PublicKey** (*string*) - La clé publique du certificat
       * **PrivateCert** (*string*) - Le certificat privé du compte
       * **PublicCert** (*string*) - Le certificat public du compte

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 163

          {
            "Name": "certificat1",
            "PrivateKey": "*clé privée*",
            "PublicKey": "*clé publique*"
            "PrivateCert": "*certificat privée*"
            "PublicCert": "*certificat public*"
          }