Consulter une instance cloud
============================

.. http:get:: /api/clouds/(string:name)

   Renvoie l'instance cloud demandée.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: L'instance cloud a été renvoyée avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: L'instance cloud demandée n'existe pas

   :resjson string name: Le nom de l'instance cloud.
   :resjson string type: Le type d'instance cloud. Voir la section
      :ref:`cloud <reference-cloud>` pour la liste des types d'instance cloud
      supportés.:resjson string key: La clé d'authentification de l'instance cloud (si
      l'instance cloud requiert une authentification).

   :resjson object options: Les options de connexion à l'instance cloud. Voir
      la section :ref:`cloud <reference-cloud>` pour avoir la liste des options
      disponibles pour le type concerné.

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/clouds/aws HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 165

         {
           "name": "aws",
           "type": "s3",
           "key": "foo",
           "options": {
               "region": "eu-west-1"
           }
         }