require 'xcodeproj'
require 'json'

project_path = '/Users/godrei/Develop/bitrise/sample-apps/ios-no-shared-schemes/BitriseXcode7Sample.xcodeproj'
begin
  project = Xcodeproj::Project.open(project_path)
  project.recreate_user_schemes
  project.save
rescue => ex
  puts(ex.inspect.to_s)
  puts('--- Stack trace: ---')
  puts(ex.backtrace.to_s)
  exit(1)
end

