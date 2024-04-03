Consulter un partenaire
=======================

.. http:get:: /api/partners/(string:partner_name)

   Renvoie les informations du partenaire portant le nom ``partner_name``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Les informations du partenaire ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :resjson string name: Le nom du partenaire
   :resjson string protocol: Le protocole utilisé par le partenaire
   :resjson string address: L'adresse du partenaire (en format [adresse:port])
   :resjson object protoConfig: La configuration du partenaire encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :resjson array authMethods: La liste des valeurs utilisées par le partenaire
      pour s'authentifier auprès de la gateway quand celle-ci s'y connecte.
   :resjson object authorizedRules: Les règles que le partenaire est autorisé à
      utiliser pour les transferts.

      * **sending** (*array* of *string*) - Les règles d'envoi.
      * **reception** (*array* of *string*) - Les règles de réception.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/partners/waarp_sftp HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 292

      {
        "name": "waarp_sftp",
        "protocol": "sftp",
        "address": "waarp.org:2022",
        "protoConfig": {},
        "authMethods": ["waarp_hostkey"],
        "authorizedRules": {
          "sending": ["règle_envoi_1", "règle_envoi_2"],
          "reception": ["règle_récep_1", "règle_récep_2"]
        }
      }
