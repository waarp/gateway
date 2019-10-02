Remplacer un partenaire
=======================

.. http:put:: /api/partners/(int:partner_id)

   Remplace le partenaire portant le numéro ``partner_id`` par celui renseigné
   en format JSON dans le corps de la requête. Les champs non-spécifiés seront
   remplacés par leur valeur par défaut.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du partenaire
   :reqjson string protocol: Le protocole utilisé par le partenaire
   :reqjson string protoConfig: La configuration du partenaire encodé dans une
      chaîne de caractères au format JSON.

   **Exemple de requête**

       .. code-block:: http

          PUT https://my_waarp_gateway.net/api/partners/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 83

          {
            "name": "waarp_sftp_new",
            "protocol": "sftp",
            "protoConfig": "{\"address\":\"waarp.fr\",\"port\":22}
          }


   **Réponse**

   :statuscode 201: Le partenaire a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du partenaire sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resheader Location: Le chemin d'accès au partenaire modifié

   :Example:
       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/partners/1