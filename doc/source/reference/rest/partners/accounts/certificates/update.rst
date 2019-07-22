Modifier un certificat
======================

.. http:patch:: /api/partners/(partner)/accounts/(account)/certificates/(certificate)

   Modifie le certificat `certificate` du compte `account` rattaché au partenaire portant
   le nom `partner` avec les informations renseignées en format JSON dans le corps
   de la requête. Les champs non-spécifiés seront remplacés par leur valeur par défaut.

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

          PUT /api/partners/partenaire1/accounts/utilisateur1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 69

          {
            "PublicKey": "*nouvelle clé publique*",
            "PublicCert": "*nouvelle certificat public*"
          }


   **Réponse**

   :statuscode 201: Le certificat a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat, compte ou partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au certificat modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/partners/partenaire1/accounts/utilisateur1/certificates/certificat1b