Modifier un certificat
======================

.. http:put:: /api/servers/(string:server)/certificates/(string:cert_name)

   Met à jour le certificat demandés à partir des informations renseignées en JSON.
   Les champs non-spécifiés resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du certificat
   :reqjson string privateKey: La clé privée du certificat
   :reqjson string publicKey: La clé publique du certificat
   :reqjson string certificate: Le certificat de l'entité

   :statuscode 201: Le certificat a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le serveur ou le certificat demandés n'existent pas

   :resheader Location: Le chemin d'accès au certificat modifié


   .. admonition:: Exemple de requête

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/servers/serveur_sftp/certificates/certificat_sftp HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 69

         {
           "name": "certificat_sftp_new",
           "privateKey": "<clé privée>",
           "publicKey": "<clé publique>",
           "cert": "<certificat>"
         }

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/servers/serveur_sftp/certificates/certtificat_sftp