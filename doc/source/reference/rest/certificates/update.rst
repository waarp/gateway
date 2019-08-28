Modifier un certificat
======================

.. http:patch:: /api/certificates/(int:certificate_id)

   Met à jour le certificat portant le numéro ``certificate_id`` avec les informations
   renseignées en format JSON dans le corps de la requête. Les champs non-spécifiés
   resteront inchangés.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson number ID: Le numéro unique du certificat
   :reqjson string Name: Le nom du certificat
   :reqjson number AccountID: Le numéro du compte auquel appartient le certificat
   :reqjson string PrivateKey: La clé privée du compte
   :reqjson string PublicKey: La clé publique du compte
   :reqjson string Cert: Le certificat de la clé publique

   **Exemple de requête**

       .. code-block:: http

          PATCH /api/certificate/1234 HTTP/1.1
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
   :statuscode 404: Le certificat demandé n'existe pas

   :resheader Location: Le chemin d'accès au certificat modifié

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/certificates/1234