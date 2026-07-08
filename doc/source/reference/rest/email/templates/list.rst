Lister les templates d'email
============================

.. http:get:: /api/email/templates

   Renvoie une liste des templates d'email connus.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les templates seront triés *(défaut: name+)*
   :type sort: [name+\|name-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resjson array templates: La liste des templates d'email demandés
   :resjsonarr string name: Le nom du template.
   :resjsonarr string subject: Le sujet de l'email.
   :resjsonarr string mimeType: Le type MIME du corps de l'email.
   :resjsonarr string body: Le corps de l'email.
   :resjsonarr array attachments: La liste des fichiers joints au template.

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/email/templates?limit=10&sort=name+ HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 165

         {
           "templates": [{
             "name": "transfer-error",
             "subject": "Erreur de transfert",
             "mimeType": "text/plain",
             "body": "Le transfert {{ .Rule }} a échoué.",
             "attachments": []
           }]
         }
