Modifier un partenaire
======================

.. http:put:: /api/partners/(int:partner_id)

   Met à jour le partenaire portant le numéro ``partner_id`` avec les informations
   renseignées en format JSON dans le corps de la requête. Les champs non-spécifiés
   resteront inchangés.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du partenaire
   :reqjson string protocol: Le protocole utilisé par le partenaire
   :reqjson object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON.

   **Exemple de requête**

       .. code-block:: http

          PATCH https://my_waarp_gateway.net/api/partners/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 105

          {
            "name": "waarp_sftp_new",
            "protocol": "sftp",
            "protoConfig": {
              "address": "waarp.fr",
              "port": 22,
              "root": "/new/sftp/root"
            }
          }


   **Réponse**

   :statuscode 201: Le partenaire a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au partenaire modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/partners/1