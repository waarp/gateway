*********************
Ajouter un certificat
*********************

.. http:post:: /api/certificates

   Ajoute un nouveau certificat rattaché au compte portant le numéro ``AccountID``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string Name: Le nom du certificat
   :reqjson number AccountID: Le numéro du compte auquel appartient le certificat
   :reqjson string PrivateKey: La clé privée du compte
   :reqjson string PublicKey: La clé publique du compte
   :reqjson string Cert: Le certificat de la clé publique

   **Exemple de requête**

       .. code-block:: http

          POST /api/certificates HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 156

          {
            "ID": 1234,
            "Name": "certificat1",
            "PartnerID": 12345,
            "PrivateKey": "*clé privée*",
            "PublicKey": "*clé publique*",
            "Cert": "*certificat*"
          }

   **Réponse**

   :statuscode 201: Le certificat a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du certificat sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau certificat créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: /api/certificates/1234
