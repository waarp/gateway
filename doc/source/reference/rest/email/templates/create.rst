Ajouter un template d'email
===========================

.. http:post:: /api/email/templates

   Ajoute un nouveau template d'email.

   :reqheader Authorization: Les identifiants de l'utilisateur REST.

   :reqjson string name: Le nom du nouveau template.
   :reqjson string subject: Le sujet de l'email.
   :reqjson string mimeType: Le type MIME du corps de l'email.
   :reqjson string body: Le corps de l'email.
   :reqjson array attachments: La liste des fichiers à joindre à l'email.

   :statuscode 201: Le template a été créé avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resheader Location: Le chemin d'accès au nouveau template créé

   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/email/templates HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 142

         {
           "name": "transfer-error",
           "subject": "Erreur de transfert",
           "mimeType": "text/plain",
           "body": "Le transfert {{ .Rule }} a échoué.",
           "attachments": []
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/email/templates/transfer-error
