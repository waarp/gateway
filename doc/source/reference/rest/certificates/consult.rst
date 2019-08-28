Consulter un certificat
=======================

.. http:get:: /api/certificates/(int:certificate_id)

   Renvoie le certificat portant le numéro ``certificate_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Ememple de requête**

       .. code-block:: http

          GET /api/partners/partenaire1/accounts/utilisateur1/certificates/certificat1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le certificat a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le certificat demandé n'existe pas

   :resjson number ID: Le numéro unique du certificat
   :resjson string Name: Le nom du certificat
   :resjson number AccountID: Le numéro du compte auquel appartient le certificat
   :resjson string PrivateKey: La clé privée du compte
   :resjson string PublicKey: La clé publique du compte
   :resjson string Cert: Le certificat de la clé publique

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 197

          {
            "ID": 1234,
            "Name": "certificat1",
            "PartnerID": 12345,
            "PrivateKey": "*clé privée*",
            "PublicKey": "*clé publique*"
            "PrivateCert": "*certificat privée*",
            "PublicCert": "*certificat public*"
          }