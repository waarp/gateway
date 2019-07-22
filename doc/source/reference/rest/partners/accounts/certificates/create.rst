Ajouter un certificat
=====================

.. http:post:: /api/partners/(partner)/accounts/(account)/certificates

   Ajoute un nouveau certificat au compte `account` rattaché au partenaire
   nommé `partner`.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Request JSON Object:

       * **Name** (*string*) - Le nom du certificat
       * **PrivateKey** (*string*) - La clé privée du certificat
       * **PublicKey** (*string*) - La clé publique du certificat
       * **PrivateCert** (*string*) - Le certificat privé du compte
       * **PublicCert** (*string*) - Le certificat public du compte

   :Example:
       .. code-block:: http

          POST /api/partners/partenaire1/accounts/utilisateur1/certificates HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 164

          {
            "Name": "certificat1",
            "PrivateKey": "*clé privée*",
            "PublicKey": "*clé publique*",
            "PrivateCert": "*certificat privée*"
            "PublicCert": "*certificat public*"
          }


   **Réponse**

   :statuscode 201: Le certificat a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du certificat sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte ou le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au nouveau certificat créé

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/partenaire1/accounts/utilisateur1/certificates/certificat1
