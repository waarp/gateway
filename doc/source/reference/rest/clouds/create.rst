Ajouter une instance cloud
==========================

.. http:post:: /api/clouds

   Ajoute une nouvelle instance cloud.

   :reqheader Authorization: Les identifiants de l'utilisateur REST.

   :reqjson string name: Le nom de la nouvelle instance cloud.
   :reqjson string type: Le type de la nouvelle instance cloud. Voir la section
      :ref:`cloud <reference-cloud>` pour la liste des types d'instance cloud
      supportés.
   :reqjson string key: La clé d'authentification de la nouvelle instance cloud
      (si l'instance cloud requiert une authentification).
   :reqjson string secret: Le secret d'authentification (mot de passe, token...)
      de la nouvelle instance cloud (si l'instance cloud requiert une
      authentification).
   :reqjson object options: Les options de connexion à la nouvelle instance
      cloud. Voir la section :ref:`cloud <reference-cloud>` pour avoir la liste
      des options disponibles pour le type concerné.

   :statuscode 201: L'instance cloud a été créée avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resheader Location: Le chemin d'accès à la nouvelle instance cloud créée

   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/clouds HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 195

         {
           "name": "aws",
           "type": "s3",
           "key": "foobar",
           "secret": "sesame",
           "options": {
             "region": "eu-west-1",
           }
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/clouds/aws